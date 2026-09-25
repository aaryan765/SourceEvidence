package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type GitReport struct {
	Repo         string   `json:"repo"`
	Head         string   `json:"head"`
	Refs         []GitRef `json:"refs"`
	Commits      int      `json:"commits"`
	Authors      []Count  `json:"authors"`
	ChangedFiles []Count  `json:"changed_files"`
}
type GitRef struct {
	Name   string `json:"name"`
	Object string `json:"object"`
}
type Count struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type gitObject struct {
	typ  string
	data []byte
}
type gitStore struct {
	repo  *Repo
	packs map[[20]byte]packLocation
}
type packLocation struct {
	path   string
	offset uint64
}

func NewGitStore(r *Repo) *gitStore { return &gitStore{repo: r, packs: map[[20]byte]packLocation{}} }

func (g *gitStore) indexPacks() error {
	pdir := filepath.Join(g.repo.Git, "objects", "pack")
	ents, err := os.ReadDir(pdir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".idx") {
			continue
		}
		idxPath := filepath.Join(pdir, e.Name())
		b, err := os.ReadFile(idxPath)
		if err != nil {
			continue
		}
		if len(b) < 8 || string(b[:4]) != "\xfftOc" {
			continue
		}
		ver := binary.BigEndian.Uint32(b[4:8])
		if ver != 2 {
			continue
		}
		pos := 8
		if len(b) < pos+256*4 {
			return nil
		}
		fan := make([]uint32, 256)
		for i := 0; i < 256; i++ {
			fan[i] = binary.BigEndian.Uint32(b[pos+i*4 : pos+i*4+4])
		}
		pos += 256 * 4
		n := int(fan[255])
		if len(b) < pos+n*20+n*4+n*4 {
			return nil
		}
		hashes := make([][20]byte, n)
		for i := 0; i < n; i++ {
			copy(hashes[i][:], b[pos+i*20:pos+i*20+20])
		}
		pos += n * 20
		pos += n * 4
		offs := make([]uint32, n)
		for i := 0; i < n; i++ {
			offs[i] = binary.BigEndian.Uint32(b[pos+i*4 : pos+i*4+4])
		}
		pos += n * 4
		largeCount := 0
		for _, o := range offs {
			if o&0x80000000 != 0 {
				largeCount++
			}
		}
		large := make([]uint64, largeCount)
		for i := range large {
			if pos+8 > len(b) {
				break
			}
			large[i] = binary.BigEndian.Uint64(b[pos : pos+8])
			pos += 8
		}
		li := 0
		packPath := strings.TrimSuffix(idxPath, ".idx") + ".pack"
		for i, h := range hashes {
			off := uint64(offs[i])
			if offs[i]&0x80000000 != 0 {
				off = large[li]
				li++
			}
			g.packs[h] = packLocation{packPath, off}
		}
	}
	return nil
}

func hexTo20(s string) ([20]byte, error) {
	var h [20]byte
	b, e := hex.DecodeString(s)
	if e != nil || len(b) != 20 {
		return h, errors.New("bad object id")
	}
	copy(h[:], b)
	return h, nil
}
func (g *gitStore) object(id [20]byte) (gitObject, error) {
	// Loose object first.
	hs := hex.EncodeToString(id[:])
	p := filepath.Join(g.repo.Git, "objects", hs[:2], hs[2:])
	if b, err := os.ReadFile(p); err == nil {
		zr, e := zlib.NewReader(bytes.NewReader(b))
		if e != nil {
			return gitObject{}, e
		}
		raw, e := io.ReadAll(zr)
		zr.Close()
		if e != nil {
			return gitObject{}, e
		}
		i := bytes.IndexByte(raw, 0)
		if i < 0 {
			return gitObject{}, errors.New("bad loose object")
		}
		typeEnd := bytes.IndexByte(raw[:i], ' ')
		if typeEnd < 0 || typeEnd >= i {
			return gitObject{}, errors.New("bad loose object header")
		}
		return gitObject{string(raw[:typeEnd]), raw[i+1:]}, nil
	}
	loc, ok := g.packs[id]
	if !ok {
		return gitObject{}, fmt.Errorf("object %s not found", hs)
	}
	return g.packObject(loc.path, loc.offset, map[uint64]bool{})
}
func (g *gitStore) packObject(path string, offset uint64, seen map[uint64]bool) (gitObject, error) {
	if seen[offset] {
		return gitObject{}, errors.New("delta cycle")
	}
	seen[offset] = true
	f, e := os.Open(path)
	if e != nil {
		return gitObject{}, e
	}
	defer f.Close()
	if _, e = f.Seek(int64(offset), io.SeekStart); e != nil {
		return gitObject{}, e
	}
	head := make([]byte, 1)
	if _, e = io.ReadFull(f, head); e != nil {
		return gitObject{}, e
	}
	c := head[0]
	typ := int((c >> 4) & 7)
	size := uint64(c & 0x0f)
	shift := uint(4)
	for c&0x80 != 0 {
		if _, e = io.ReadFull(f, head); e != nil {
			return gitObject{}, e
		}
		c = head[0]
		size |= uint64(c&0x7f) << shift
		shift += 7
	}
	switch typ {
	case 1, 2, 3, 4, 5:
		zr, e := zlib.NewReader(f)
		if e != nil {
			return gitObject{}, e
		}
		data, e := io.ReadAll(zr)
		zr.Close()
		if e != nil {
			return gitObject{}, e
		}
		if uint64(len(data)) != size && typ != 4 {
			return gitObject{}, errors.New("pack size mismatch")
		}
		names := []string{"", "commit", "tree", "blob", "tag", "reserved"}
		return gitObject{names[typ], data}, nil
	case 7:
		var base [20]byte
		if _, e = io.ReadFull(f, base[:]); e != nil {
			return gitObject{}, e
		}
		zr, e := zlib.NewReader(f)
		if e != nil {
			return gitObject{}, e
		}
		delta, e := io.ReadAll(zr)
		zr.Close()
		if e != nil {
			return gitObject{}, e
		}
		baseObj, e := g.object(base)
		if e != nil {
			return gitObject{}, e
		}
		data, e := applyDelta(baseObj.data, delta)
		if e != nil {
			return gitObject{}, e
		}
		return gitObject{baseObj.typ, data}, nil
	case 6:
		// OFS_DELTA: variable length encoding. The offset points backwards from current object offset.
		var b0 [1]byte
		if _, e = io.ReadFull(f, b0[:]); e != nil {
			return gitObject{}, e
		}
		c = b0[0]
		var dist uint64
		dist = uint64(c & 0x7f)
		for c&0x80 != 0 {
			var nb [1]byte
			if _, e = io.ReadFull(f, nb[:]); e != nil {
				return gitObject{}, e
			}
			c = nb[0]
			dist = (dist+1)<<7 | uint64(c&0x7f)
		}
		baseOffset := offset - dist
		zr, e := zlib.NewReader(f)
		if e != nil {
			return gitObject{}, e
		}
		delta, e := io.ReadAll(zr)
		zr.Close()
		if e != nil {
			return gitObject{}, e
		}
		baseObj, e := g.packObject(path, baseOffset, seen)
		if e != nil {
			return gitObject{}, e
		}
		data, e := applyDelta(baseObj.data, delta)
		if e != nil {
			return gitObject{}, e
		}
		return gitObject{baseObj.typ, data}, nil
	default:
		return gitObject{}, fmt.Errorf("unsupported git pack object type %d", typ)
	}
}
func readVarInt(b []byte, pos *int) (uint64, error) {
	var n uint64
	var shift uint
	for {
		if *pos >= len(b) {
			return 0, io.ErrUnexpectedEOF
		}
		c := b[*pos]
		*pos++
		n |= uint64(c&0x7f) << shift
		if c&0x80 == 0 {
			return n, nil
		}
		shift += 7
		if shift > 63 {
			return 0, errors.New("delta varint overflow")
		}
	}
}
func applyDelta(base, delta []byte) ([]byte, error) {
	p := 0
	bs, err := readVarInt(delta, &p)
	if err != nil {
		return nil, err
	}
	if bs != uint64(len(base)) {
		return nil, errors.New("delta base size mismatch")
	}
	outSize, err := readVarInt(delta, &p)
	if err != nil {
		return nil, err
	}
	if outSize > uint64(^uint(0)>>1) {
		return nil, errors.New("delta result too large")
	}
	out := make([]byte, 0, int(outSize))
	for p < len(delta) {
		op := delta[p]
		p++
		if op&0x80 == 0 {
			if op == 0 {
				return nil, errors.New("invalid delta insert opcode")
			}
			if p+int(op) > len(delta) {
				return nil, io.ErrUnexpectedEOF
			}
			out = append(out, delta[p:p+int(op)]...)
			p += int(op)
			continue
		}
		var off, size uint32
		readByte := func() (byte, error) {
			if p >= len(delta) {
				return 0, io.ErrUnexpectedEOF
			}
			v := delta[p]
			p++
			return v, nil
		}
		if op&1 != 0 {
			v, e := readByte()
			if e != nil {
				return nil, e
			}
			off |= uint32(v)
		}
		if op&2 != 0 {
			v, e := readByte()
			if e != nil {
				return nil, e
			}
			off |= uint32(v) << 8
		}
		if op&4 != 0 {
			v, e := readByte()
			if e != nil {
				return nil, e
			}
			off |= uint32(v) << 16
		}
		if op&8 != 0 {
			v, e := readByte()
			if e != nil {
				return nil, e
			}
			off |= uint32(v) << 24
		}
		if op&0x10 != 0 {
			v, e := readByte()
			if e != nil {
				return nil, e
			}
			size |= uint32(v)
		}
		if op&0x20 != 0 {
			v, e := readByte()
			if e != nil {
				return nil, e
			}
			size |= uint32(v) << 8
		}
		if op&0x40 != 0 {
			v, e := readByte()
			if e != nil {
				return nil, e
			}
			size |= uint32(v) << 16
		}
		if size == 0 {
			size = 0x10000
		}
		end := uint64(off) + uint64(size)
		if end > uint64(len(base)) {
			return nil, errors.New("delta copy outside base")
		}
		out = append(out, base[int(off):int(end)]...)
		if uint64(len(out)) > outSize {
			return nil, errors.New("delta output exceeds declared size")
		}
	}
	if uint64(len(out)) != outSize {
		return nil, errors.New("delta size mismatch")
	}
	return out, nil
}

func (g *gitStore) refs() ([]GitRef, error) {
	var refs []GitRef
	head, err := g.repo.readText("HEAD")
	if err == nil {
		h := strings.TrimSpace(head)
		obj := ""
		if strings.HasPrefix(h, "ref: ") {
			obj, _ = g.resolveRef(strings.TrimSpace(strings.TrimPrefix(h, "ref: ")))
		} else {
			obj = h
		}
		if obj != "" {
			refs = append(refs, GitRef{"HEAD", obj})
		}
	}
	var walk func(string) error
	walk = func(dir string) error {
		ents, e := os.ReadDir(filepath.Join(g.repo.Git, dir))
		if e != nil {
			return nil
		}
		for _, en := range ents {
			rel := filepath.Join(dir, en.Name())
			if en.IsDir() {
				walk(rel)
				continue
			}
			b, e := os.ReadFile(filepath.Join(g.repo.Git, rel))
			if e == nil {
				refs = append(refs, GitRef{filepath.ToSlash(rel), strings.TrimSpace(string(b))})
			}
		}
		return nil
	}
	walk("refs")
	if b, e := os.ReadFile(filepath.Join(g.repo.Git, "packed-refs")); e == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "^") {
				continue
			}
			f := strings.Fields(line)
			if len(f) == 2 {
				refs = append(refs, GitRef{f[1], f[0]})
			}
		}
	}
	uniq := map[string]GitRef{}
	for _, x := range refs {
		uniq[x.Name] = x
	}
	refs = refs[:0]
	for _, x := range uniq {
		refs = append(refs, x)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Name < refs[j].Name })
	return refs, nil
}
func (g *gitStore) resolveRef(ref string) (string, error) {
	if b, e := os.ReadFile(filepath.Join(g.repo.Git, filepath.FromSlash(ref))); e == nil {
		return strings.TrimSpace(string(b)), nil
	}
	if b, e := os.ReadFile(filepath.Join(g.repo.Git, "packed-refs")); e == nil {
		for _, line := range strings.Split(string(b), "\n") {
			f := strings.Fields(line)
			if len(f) == 2 && f[1] == ref {
				return f[0], nil
			}
		}
	}
	return "", errors.New("ref not found")
}

func parseCommit(data []byte) (tree string, parents []string, author string) {
	s := string(data)
	parts := strings.SplitN(s, "\n\n", 2)
	lines := strings.Split(parts[0], "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "tree ") {
			tree = strings.TrimSpace(strings.TrimPrefix(l, "tree "))
		}
		if strings.HasPrefix(l, "parent ") {
			parents = append(parents, strings.TrimSpace(strings.TrimPrefix(l, "parent ")))
		}
		if strings.HasPrefix(l, "author ") {
			a := strings.TrimPrefix(l, "author ")
			if i := strings.Index(a, " <"); i > 0 {
				author = a[:i]
			}
		}
	}
	return
}

type treeEntry struct {
	name string
	mode string
	id   [20]byte
	dir  bool
}

func (g *gitStore) readTree(idstr string, prefix string, out map[string][20]byte) error {
	id, e := hexTo20(idstr)
	if e != nil {
		return e
	}
	obj, e := g.object(id)
	if e != nil {
		return e
	}
	d := obj.data
	for len(d) > 0 {
		i := bytes.IndexByte(d, 0)
		if i < 0 || i+21 > len(d) {
			return errors.New("bad tree")
		}
		meta := string(d[:i])
		parts := strings.SplitN(meta, " ", 2)
		if len(parts) != 2 {
			return errors.New("bad tree entry")
		}
		name := parts[1]
		copyID := [20]byte{}
		copy(copyID[:], d[i+1:i+21])
		d = d[i+21:]
		if strings.HasSuffix(parts[0], "40000") {
			child := hex.EncodeToString(copyID[:])
			if e := g.readTree(child, filepath.ToSlash(filepath.Join(prefix, name)), out); e != nil {
				return e
			}
		} else {
			out[filepath.ToSlash(filepath.Join(prefix, name))] = copyID
		}
	}
	return nil
}

const gitHistoryLimit = 500

const gitHistoryScopeLabel = "all discovered refs (bounded to 500 commits)"

func AnalyzeGit(r *Repo) (GitReport, error) {
	g := NewGitStore(r)
	if e := g.indexPacks(); e != nil {
		return GitReport{}, e
	}
	refs, e := g.refs()
	if e != nil {
		return GitReport{}, e
	}
	out := GitReport{Repo: r.Root, Refs: refs}
	for _, x := range refs {
		if x.Name == "HEAD" {
			out.Head = x.Object
			break
		}
	}
	authors, files, commits := analyzeGitHistory(g, refs, gitHistoryLimit)
	out.Commits = commits
	out.Authors = sortCounts(authors, 15)
	out.ChangedFiles = sortCounts(files, 30)
	return out, nil
}

// analyzeGitHistory is the single bounded history walker used by both Git
// archaeology and file-level Explain/Hotspots. Keeping one traversal strategy
// prevents the same path from producing different counts in different views.
func analyzeGitHistory(g *gitStore, refs []GitRef, limit int) (map[string]int, map[string]int, int) {
	authors := map[string]int{}
	files := map[string]int{}
	seen := map[string]bool{}
	queue := []string{}
	seenRefs := map[string]bool{}
	for _, ref := range refs {
		if ref.Object == "" || seenRefs[ref.Object] {
			continue
		}
		seenRefs[ref.Object] = true
		queue = append(queue, ref.Object)
	}
	commits := 0
	for len(queue) > 0 && commits < limit {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		oid, e := hexTo20(id)
		if e != nil {
			continue
		}
		obj, e := g.object(oid)
		if e != nil || obj.typ != "commit" {
			continue
		}
		tree, parents, author := parseCommit(obj.data)
		commits++
		authors[author]++
		cur := map[string][20]byte{}
		if tree != "" {
			_ = g.readTree(tree, "", cur)
		}
		if len(parents) == 0 {
			for f := range cur {
				files[f]++
			}
		} else {
			pid := parents[0]
			poid, er := hexTo20(pid)
			if er == nil {
				pobj, er := g.object(poid)
				if er == nil && pobj.typ == "commit" {
					pt, _, _ := parseCommit(pobj.data)
					prev := map[string][20]byte{}
					_ = g.readTree(pt, "", prev)
					for f, h := range cur {
						if ph, ok := prev[f]; !ok || ph != h {
							files[f]++
						}
					}
					for f := range prev {
						if _, ok := cur[f]; !ok {
							files[f]++
						}
					}
				}
			}
		}
		for _, parent := range parents {
			queue = append(queue, parent)
		}
	}
	return authors, files, commits
}

// GitChangesForPath counts changes using the exact same bounded repository
// history traversal as Hotspots. The scope is intentionally explicit so users
// do not have to interpret two different numbers for the same file.
func GitChangesForPath(r *Repo, rel string) (int, error) {
	g := NewGitStore(r)
	if err := g.indexPacks(); err != nil {
		return 0, err
	}
	refs, err := g.refs()
	if err != nil {
		return 0, err
	}
	_, files, _ := analyzeGitHistory(g, refs, gitHistoryLimit)
	path := filepath.ToSlash(filepath.Clean(rel))
	return files[path], nil
}

func sortCounts(m map[string]int, n int) []Count {
	out := make([]Count, 0, len(m))
	for k, v := range m {
		out = append(out, Count{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Name < out[j].Name
		}
		return out[i].Count > out[j].Count
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}
func gitHeadName(r *Repo) string {
	b, e := r.readText("HEAD")
	if e != nil {
		return ""
	}
	s := strings.TrimSpace(b)
	if strings.HasPrefix(s, "ref: ") {
		return strings.TrimPrefix(s, "ref: ")
	}
	return "detached"
}
