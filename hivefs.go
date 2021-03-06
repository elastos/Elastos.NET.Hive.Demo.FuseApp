// hivefs implements a  "hive cluster" fuse client file system.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"github.com/elastos/Elastos.NET.Hive.Demo.FuseApp/hive"
	"github.com/google/logger"
	"golang.org/x/net/context"
)

type hiveConfig struct {
	host       string
	port       int
	uid        string
	mountpoint string
}

var config hiveConfig

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -host <host> -port <port number> -uid <uid> MOUNTPOINT\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "For example:\n")
	fmt.Fprintf(os.Stderr, "  %s -host 127.0.0.1 -port 9095 -uid uid-ee978fa7-18b6-43d4-9f3e-3e6562131036 MOUNTPOINT\n", os.Args[0])
}

func initArgs() {
	const (
		defaultHost = "127.0.0.1"
		usageHost   = "the one of hive cluster address"
		defaultPort = 9095
		usagePort   = "hive port"
	)
	flag.StringVar(&config.host, "host", defaultHost, usageHost)
	flag.IntVar(&config.port, "port", defaultPort, usagePort)
	flag.StringVar(&config.uid, "uid", "", "uid")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 || config.uid == "" {
		usage()
		os.Exit(2)
	}

	config.mountpoint = flag.Arg(flag.NArg() - 1)
}

func inodeFromPath(path string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(path))
	return h.Sum64()
}

func main() {
	defer logger.Init("client", true, false, ioutil.Discard).Close()
	logger.SetFlags(log.Lshortfile)

	initArgs()

	hive, err := hive.NewConnector(fmt.Sprintf("%s:%d", config.host, config.port))
	if err != nil {
		log.Fatal(err)
	}

	c, err := fuse.Mount(
		config.mountpoint,
		fuse.FSName("hive"),
		fuse.Subtype("hivefs"),
		fuse.LocalVolume(),
		fuse.VolumeName("hive"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	err = fs.Serve(c, &FS{connector: hive})
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

// FS implements the hive file system.
type FS struct {
	connector *hive.Connector
	Node      fs.Node
}

var _ fs.FS = (*FS)(nil)

func (fs *FS) Root() (fs.Node, error) {
	n := &Dir{
		connector: fs.connector, entries: nil, path: "/",
	}
	return n, nil
}

type Dir struct {
	connector *hive.Connector
	path      string
	entries   *[]fuse.Dirent
}

var (
	_ fs.Node               = (*Dir)(nil)
	_ fs.HandleReadDirAller = (*Dir)(nil)
	_ fs.NodeMkdirer        = (*Dir)(nil)
	_ fs.NodeRemover        = (*Dir)(nil)
	_ fs.NodeCreater        = (*Dir)(nil)
	_ fs.NodeRenamer        = (*Dir)(nil)
	_ fs.NodeOpener         = (*Dir)(nil)
)

func (dir *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	logger.Info("Attr: ", dir.path)

	if dir.path == "" {
		a.Inode = 1
		a.Mode = os.ModeDir | 0755
		return nil
	}

	stat, err := dir.connector.FilesStat(config.uid, dir.path)
	if err != nil {
		return fuse.ENOENT
	}

	if stat.Type == "file" {
		// a.Inode = inodeFromPath(path)
		a.Size = stat.Size
		a.Mode = 0666
	} else if stat.Type == "directory" {
		// assume dir stat is OK
		// a.Inode = inodeFromPath(dir.path)
		a.Mode = os.ModeDir | 0755
	}

	return nil
}

func (dir *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	path := filepath.Join(dir.path, name)

	logger.Info("Lookup: ", path)

	filesStat, err := dir.connector.FilesStat(config.uid, path)
	if err != nil {
		return nil, fuse.ENOENT
	}

	logger.Info("Lookup: filesStat", filesStat)

	if filesStat.Type == "file" {
		return &File{connector: dir.connector, parent: dir, path: path, size: filesStat.Size}, nil
	}

	if filesStat.Type == "directory" {
		return &Dir{connector: dir.connector, path: path}, nil
	}

	// if dir.entries != nil {
	// 	for _, v := range *dir.entries {

	// 		logger.Info("Lookup: ", v.Name, " name:", name)
	// 		if v.Name == name {
	// 			path := filepath.Join(dir.path, name)

	// 			if v.Type == fuse.DT_Dir {
	// 				return &Dir{connector: dir.connector, path: path}, nil
	// 			} else if v.Type == fuse.DT_File {
	// 				return &File{connector: dir.connector, parent: dir, path: path}, nil
	// 			}

	// 			break
	// 		}
	// 	}
	// }

	return nil, fuse.ENOENT
}

func (dir *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	logger.Info("ReadDirAll: ", dir.path)

	path := dir.path

	dirents := []fuse.Dirent{}

	dirs, err := dir.connector.FilesLs(config.uid, path)
	if err != nil {
		return dirents, err
	}

	// append dummy dir: .. and .
	dirents = append(dirents, fuse.Dirent{Name: "..", Type: fuse.DT_Dir})
	dirents = append(dirents, fuse.Dirent{Name: ".", Type: fuse.DT_Dir})

	for _, v := range dirs.Entries {
		var dirent = fuse.Dirent{}
		// dirent.Inode = inodeFromPath(path + "/" + v.Name)
		dirent.Name = v.Name
		if v.Type == 0 {
			dirent.Type = fuse.DT_Dir
		} else {
			dirent.Type = fuse.DT_File
		}

		dirents = append(dirents, dirent)
	}

	dir.entries = &dirents

	return dirents, nil
}

func (dir *Dir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {

	path := filepath.Join(dir.path, req.Name)

	logger.Info("Mkdir: ", path)

	err := dir.connector.FilesMkdir(config.uid, path, false)

	if err != nil {
		return nil, err
	}

	d := &Dir{connector: dir.connector, path: path}
	return d, nil
}

func (dir *Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	path := filepath.Join(dir.path, req.Name)

	logger.Info("Remove: ", path)

	err := dir.connector.FilesRm(config.uid, path, true)

	return err
}

func (dir *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	path := filepath.Join(dir.path, req.Name)

	logger.Info("Create: ", path)

	if req.Flags&fuse.OpenReadOnly != 0 {
		return dir, &File{connector: dir.connector, path: path, flags: req.Flags}, nil
	}

	if req.Flags&fuse.OpenAppend != 0 {
		// TODO:
	}

	if req.Flags&fuse.OpenCreate != 0 {
		// TODO:
	}

	if req.Flags&fuse.OpenTruncate != 0 {
		// TODO:
	}

	if req.Flags&fuse.OpenDirectory != 0 {
		logger.Info("Use mkdir to create dir: ", path)
		return nil, nil, fuse.ENOENT
	}

	err := dir.connector.FilesWrite(config.uid, path, 0, true, false, 0, &bytes.Buffer{})
	if err != nil {
		// record error but omit it
		logger.Error("Create: ", path, "error: ", err)
	}

	// // assume file has create successfully
	// var dirent = fuse.Dirent{}
	// // dirent.Inode = inodeFromPath(path + "/" + v.Name)
	// dirent.Name = req.Name
	// dirent.Type = fuse.DT_File

	// *dir.entries = append(*dir.entries, dirent)
	node := &File{connector: dir.connector, path: path, flags: req.Flags}
	handle := node

	return node, handle, nil
}

func (dir *Dir) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	newDir2, ok := newDir.(*Dir)
	if !ok {
		return fuse.EIO
	}

	oldPath := filepath.Join(dir.path, req.OldName)
	newPath := filepath.Join(newDir2.path, req.NewName)

	logger.Info("file Rename: ", oldPath, " -> ", newPath)

	err := dir.connector.FilesMv(config.uid, oldPath, newPath)
	if err == nil {
		// oldEntries := *dir.entries
		// newEntries := *newDir2.entries
		// for k, dirent := range oldEntries {
		// 	if dirent.Name == req.OldName {
		// 		newEntries = append(newEntries, dirent)
		// 		newDir2.entries = &newEntries

		// 		oldEntries = append(oldEntries[:k], oldEntries[k+1:]...)
		// 		dir.entries = &oldEntries

		// 		break
		// 	}
		// }
	}

	return err
}

func (dir *Dir) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	logger.Info("Open: ", dir.path)

	if req.Dir {
		return &Dir{connector: dir.connector, path: dir.path}, nil
	} else {
		return &File{connector: dir.connector, path: dir.path}, nil
	}
}

// File implements both Node and Handle for the hello file.
type File struct {
	connector *hive.Connector
	parent    *Dir
	path      string
	size      uint64
	offset    uint64
	flags     fuse.OpenFlags
}

var (
	_ fs.Node            = (*File)(nil)
	_ fs.Handle          = (*File)(nil)
	_ fs.HandleReadAller = (*File)(nil)
	_ fs.HandleWriter    = (*File)(nil)
	_ fs.NodeOpener      = (*File)(nil)
)

func (file *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	logger.Info("file Open: ", file.path)

	if req.Dir {
		return &Dir{connector: file.connector, path: file.path}, nil
	} else {

		return &File{connector: file.connector, path: file.path}, nil
	}
}

func (file *File) Attr(ctx context.Context, a *fuse.Attr) error {
	logger.Info("file Attr: ", file.path)

	path := file.path
	if path == "" {
		path = "/"
	}

	stat, err := file.connector.FilesStat(config.uid, path)
	if err != nil {
		// found := false
		// // assume file has created
		// for _, v := range *file.parent.entries {
		// 	vpath := filepath.Join(file.parent.path, v.Name)
		// 	if vpath == path {
		// 		found = true
		// 		break
		// 	}
		// }
		// if found {
		// 	a.Size = 0
		// 	a.Mode = 0666
		// } else {
		// 	return err
		// }
		return err
	}

	// a.Inode = inodeFromPath(path)
	a.Size = stat.Size
	a.Mode = 0666

	return nil
}

var _ = fs.NodeOpener(&File{})

func (file *File) ReadAll(ctx context.Context) ([]byte, error) {
	logger.Info("file ReadAll: ", file.path)

	path := file.path
	data, err := file.connector.FilesRead(config.uid, path)

	return data, err
}

func (file *File) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	logger.Info("file Flush: ", file.path)

	return nil
}

func (file *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	logger.Info("file Write: ", file.path)

	buf := bytes.NewBuffer(req.Data)
	truncate := true
	if req.Offset > 0 {
		truncate = false
	}
	err := file.connector.FilesWrite(config.uid, file.path, req.Offset, true, truncate, len(req.Data), buf)
	if err != nil {
		// record error but omit it
		logger.Error("Write: ", file.path, "error: ", err)
	}
	resp.Size = len(req.Data)

	return nil
}
