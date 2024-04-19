// Package fileloader
/*
 目录树实现，返回数据包括全路径、节点名、是否目录、扩展名，子节点等
*/
package fileloader

import (
	"fireboom-server/pkg/plugins/i18n"
	"github.com/spf13/cast"
	"golang.org/x/exp/slices"
	"io/fs"
	"path/filepath"
	"strings"
)

type (
	DataTree struct {
		Name      string    `json:"name"`
		Path      string    `json:"path"`
		IsDir     bool      `json:"isDir"`
		Extension string    `json:"extension,omitempty"`
		Items     DataTrees `json:"items,omitempty"`
		Extra     any       `json:"extra,omitempty"`
	}
	DataTrees []*DataTree
)

// GetDataTrees 返回目录树结构数据，仅multipleRW支持目录树结构
// 返回数据会按照目录优先，名称其次的顺序对结果排序(每层均有序)
func (p *Model[T]) GetDataTrees() (trees DataTrees, err error) {
	if p.rwType != multipleRW {
		err = i18n.NewCustomErrorWithMode(p.modelName, nil, i18n.LoaderMultipleOnlyError)
		return
	}

	dirTreeMap := make(map[string]*DataTree)
	_ = filepath.Walk(p.Root, func(path string, info fs.FileInfo, err error) error {
		if info == nil || path == p.Root {
			return nil
		}

		basename := filepath.Base(path)
		current := &DataTree{Name: basename}
		current.Path, _ = filepath.Rel(p.Root, path)
		current.Path = filepath.ToSlash(current.Path)
		parent, parentExist := dirTreeMap[filepath.Dir(path)]
		var itemAppendIgnore bool
		defer func() {
			if itemAppendIgnore {
				return
			}

			if parentExist {
				parent.Items = append(parent.Items, current)
			} else {
				trees = append(trees, current)
			}
		}()

		if info.IsDir() {
			current.IsDir = true
			dirTreeMap[path] = current
			return nil
		}

		extension := string(p.Extension)
		if !strings.HasSuffix(path, extension) {
			itemAppendIgnore = true
			return nil
		}

		dataName := strings.TrimSuffix(current.Path, extension)
		data, err := p.GetByDataName(dataName)
		if err != nil || !p.filter(data) {
			itemAppendIgnore = true
			return nil
		}

		current.Name = strings.TrimSuffix(current.Name, extension)
		current.Path = strings.TrimSuffix(current.Path, extension)
		current.Extension = extension
		if extra := p.DataTreeExtra; extra != nil {
			current.Extra = extra(data)
		}

		return nil
	})
	trees.sort()
	return
}

func (d DataTrees) sort() {
	if len(d) == 0 {
		return
	}

	slices.SortFunc(d, func(a, b *DataTree) bool {
		aInt, bInt := cast.ToInt(!a.IsDir), cast.ToInt(!b.IsDir)

		return aInt < bInt || aInt == bInt && a.Name < b.Name
	})
	for _, tree := range d {
		tree.Items.sort()
	}
}
