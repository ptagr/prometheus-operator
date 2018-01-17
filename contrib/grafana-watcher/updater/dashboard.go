// Copyright 2016 The prometheus-operator Authors
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

package updater

import (
	"log"
	"os"
	"path/filepath"

	"github.com/coreos/prometheus-operator/contrib/grafana-watcher/grafana"
	"bytes"
	"encoding/json"
)

type Updater interface {
	Init() error
	OnModify() error
}

type GrafanaDashboardUpdater struct {
	client grafana.DashboardsInterface
	globs  []string
	ignoreMissing bool
}

func NewGrafanaDashboardUpdater(c grafana.DashboardsInterface, g []string, i bool) Updater {
	return &GrafanaDashboardUpdater{
		client: c,
		globs:  g,
		ignoreMissing: i,
	}
}

func (u *GrafanaDashboardUpdater) Init() error {
	return u.updateDashboards()
}

func (u *GrafanaDashboardUpdater) OnModify() error {
	return u.updateDashboards()
}

func (u *GrafanaDashboardUpdater) updateDashboards() error {

	dashboards := make([]string, 0)
	if u.ignoreMissing {
		err := u.loadDashboardsFromFiles(&dashboards)
		if err != nil {
			return err
		}
	}

	err := u.deleteAllDashboards(dashboards)
	if err != nil {
		return err
	}

	err = u.createDashboardsFromFiles()
	if err != nil {
		return err
	}

	return nil
}

func (u *GrafanaDashboardUpdater) deleteAllDashboards(dashboards []string) error {
	log.Println("Retrieving existing dashboards")
	searchResults, err := u.client.Search()
	if err != nil {
		return err
	}


	log.Printf("Deleting %d dashboards\n", len(searchResults))
	for _, d := range searchResults {
		if u.ignoreMissing && contains(dashboards, d.Title) {
			log.Println("Ignoring dashboard:", d.Slug())
		} else {
			log.Println("Deleting dashboard:", d.Slug())
		}

		err := u.client.Delete(d.Slug())
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *GrafanaDashboardUpdater) createDashboardsFromFiles() error {
	for _, glob := range u.globs {
		filePaths, err := filepath.Glob(filepath.Join(glob, "*-dashboard.json"))
		if err != nil {
			return err
		}

		for _, fp := range filePaths {
			err = u.createDashboardFromFile(fp)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (u *GrafanaDashboardUpdater) loadDashboardsFromFiles(dashboards *[]string) error {
	for _, glob := range u.globs {
		filePaths, err := filepath.Glob(filepath.Join(glob, "*-dashboard.json"))
		if err != nil {
			return err
		}

		for _, fp := range filePaths {
			f, err := os.Open(fp)
			if err != nil {
				return err
			}
			defer f.Close()

			buf := new(bytes.Buffer)
			buf.ReadFrom(f)

			dashboard := new(grafana.GrafanaDashboard)
			err = json.Unmarshal(buf.Bytes(), &dashboard)

			if err != nil {
				return err
			}
			*dashboards = append(*dashboards, dashboard.Title)
		}
	}

	return nil
}

func (u *GrafanaDashboardUpdater) createDashboardFromFile(filePath string) error {
	log.Println("Creating dashboard from:", filePath)

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	return u.client.Create(f)
}
