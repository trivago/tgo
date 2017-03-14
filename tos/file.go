// Copyright 2015-2016 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tos

import (
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
)

// Chown is a wrapper around ChownId that allows changing user and group by name.
func Chown(filePath, usr, grp string) error {
	var uid, gid int

	if userInfo, err := user.Lookup(usr); err != nil {
		return err
	} else if uid, err = strconv.Atoi(userInfo.Uid); err != nil {
		return err
	}

	if groupInfo, err := user.LookupGroup(grp); err != nil {
		return err
	} else if gid, err = strconv.Atoi(groupInfo.Gid); err != nil {
		return err
	}

	return ChownId(filePath, uid, gid)
}

// ChownId is a wrapper around os.Chown that allows changing user and group
// recursively if given a directory.
func ChownId(filePath string, uid, gid int) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		files, err := ioutil.ReadDir(filePath)
		if err != nil {
			return err
		}
		for _, file := range files {
			if err := ChownId(filePath+"/"+file.Name(), uid, gid); err != nil {
				return err
			}
		}
	}

	return os.Chown(filePath, uid, gid)
}

// Chmod is a wrapper around os.Chmod that allows changing rights recursively
// if a directory is given.
func Chmod(filePath string, mode os.FileMode) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		files, err := ioutil.ReadDir(filePath)
		if err != nil {
			return err
		}
		for _, file := range files {
			if err := Chmod(filePath+"/"+file.Name(), mode); err != nil {
				return err
			}
		}
	}

	return os.Chmod(filePath, mode)
}

// Copy is a file copy helper. Files will be copied to their destination,
// overwriting existing files. Already existing files that are not part of the
// copy process will not be touched. If source is a directory it is walked
// recursively. Non-existing folders in dest will be created.
// Copy returns upon the first error encountered. In-between results will not
// be rolled back.
func Copy(dest, source string) error {
	srcStat, err := os.Stat(source)
	if err != nil {
		return err
	}

	if srcStat.IsDir() {
		files, err := ioutil.ReadDir(source)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(dest, srcStat.Mode()); err != nil && !os.IsExist(err) {
			return err
		}

		for _, file := range files {
			if err := Copy(dest+"/"+file.Name(), source+"/"+file.Name()); err != nil {
				return err
			}
		}
		return nil
	}

	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return os.Chmod(dest, srcStat.Mode())
}

// Remove is a wrapper around os.Remove and allows to recursively remove
// directories and files.
func Remove(name string) error {
	stat, err := os.Stat(name)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		files, err := ioutil.ReadDir(name)
		if err != nil {
			return err
		}

		for _, file := range files {
			if err := Remove(name + "/" + file.Name()); err != nil {
				return err
			}
		}
	}

	return os.Remove(name)
}
