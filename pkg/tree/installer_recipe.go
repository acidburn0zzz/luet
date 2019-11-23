// Copyright © 2019 Ettore Di Giacinto <mudler@gentoo.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.

// InstallerRecipe is a builder imeplementation.

// It reads a Tree and spit it in human readable form (YAML), called recipe,
// It also loads a tree (recipe) from a YAML (to a db, e.g. BoltDB), allowing to query it
// with the solver, using the package object.
package tree

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mudler/luet/pkg/helpers"
	pkg "github.com/mudler/luet/pkg/package"
	"github.com/pkg/errors"
)

const (
	FinalizerFile = "finalize.yaml"
)

func NewInstallerRecipe(db pkg.PackageDatabase) Builder {
	tree := NewDefaultTree()
	tree.SetPackageSet(db)
	return &InstallerRecipe{Database: db, PackageTree: tree}
}

// InstallerRecipe is the "general" reciper for Trees
type InstallerRecipe struct {
	PackageTree pkg.Tree
	SourcePath  string
	Database    pkg.PackageDatabase
}

func (r *InstallerRecipe) Save(path string) error {

	for _, pid := range r.PackageTree.GetPackageSet().GetPackages() {

		p, err := r.PackageTree.GetPackageSet().GetPackage(pid)
		if err != nil {
			return err
		}
		dir := filepath.Join(path, p.GetCategory(), p.GetName(), p.GetVersion())
		os.MkdirAll(dir, os.ModePerm)
		data, err := p.Yaml()
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dir, DefinitionFile), data, 0644)
		if err != nil {
			return err
		}
		// Instead of rdeps, have a different tree for build deps.
		finalizerPath := p.Rel(FinalizerFile)
		if helpers.Exists(finalizerPath) { // copy finalizer file from the source tree
			helpers.CopyFile(finalizerPath, filepath.Join(dir, FinalizerFile))
		}

	}
	return nil
}

func (r *InstallerRecipe) Load(path string) error {

	if r.Tree() == nil {
		r.PackageTree = NewDefaultTree()
	}

	// tmpfile, err := ioutil.TempFile("", "luet")
	// if err != nil {
	// 	return err
	// }
	r.Tree().SetPackageSet(r.Database)
	r.SourcePath = path

	//r.Tree().SetPackageSet(pkg.NewBoltDatabase(tmpfile.Name()))
	// TODO: Handle cleaning after? Cleanup implemented in GetPackageSet().Clean()

	// the function that handles each file or dir
	var ff = func(currentpath string, info os.FileInfo, err error) error {

		if info.Name() != DefinitionFile {
			return nil // Skip with no errors
		}

		dat, err := ioutil.ReadFile(currentpath)
		if err != nil {
			return errors.Wrap(err, "Error reading file "+currentpath)
		}
		pack, err := pkg.DefaultPackageFromYaml(dat)
		if err != nil {
			return errors.Wrap(err, "Error reading yaml "+currentpath)
		}

		// Path is set only internally when tree is loaded from disk
		pack.SetPath(filepath.Dir(currentpath))
		_, err = r.Tree().GetPackageSet().CreatePackage(&pack)
		if err != nil {
			return errors.Wrap(err, "Error creating package "+pack.GetName())
		}

		return nil
	}

	err := filepath.Walk(path, ff)
	if err != nil {
		return err
	}
	return nil
}

func (r *InstallerRecipe) Tree() pkg.Tree        { return r.PackageTree }
func (r *InstallerRecipe) WithTree(t pkg.Tree)   { r.PackageTree = t }
func (r *InstallerRecipe) GetSourcePath() string { return r.SourcePath }
