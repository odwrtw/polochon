package main

import (
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/papi"
	log "github.com/sirupsen/logrus"
)

func (pfs *polochonfs) createDirNode(parent *node, name string, times time.Time) *node {
	node := parent.getChild(name)
	if node == nil {
		node = newNodeDir(name, times)
		parent.addChild(node)
	}
	node.valid = true
	return node
}

func (pfs *polochonfs) createFileNode(parent *node, res papi.Downloadable, name string,
	size int64, times time.Time) error {
	if res == nil {
		return nil
	}

	url, err := pfs.client.DownloadURL(res)
	if err != nil {
		return err
	}

	return pfs.addOrUpdateFileNode(parent, name, uint64(size), times, url)
}

func (pfs *polochonfs) addOrUpdateFileNode(parent *node, name string, size uint64, times time.Time, url string) error {
	fileNode := parent.getChild(name)
	if fileNode == nil {
		fileNode = newNodeFile(name)
		fileNode.setURL(url)
		parent.addChild(fileNode)
	}
	fileNode.size = size
	fileNode.times = times
	fileNode.valid = true
	return nil
}

func (pfs *polochonfs) createFilesNodes(parent *node, files []*papi.File, times time.Time) {
	for _, file := range files {
		if file == nil {
			continue
		}

		err := pfs.createFileNode(parent, file, file.Name, file.Size, times)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"name":  file.Name,
			}).Error("Failed to file node")
			continue
		}
	}
}

func (pfs *polochonfs) createSubtitlesNodes(parent *node, videoPath string, subs []*papi.Subtitle, times time.Time) {
	for _, sub := range subs {
		if sub.Embedded {
			continue
		}

		path := polochon.NewFile(videoPath).SubtitlePath(sub.Lang)
		err := pfs.createFileNode(parent, sub, path, sub.Size, times)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"video": videoPath,
				"lang":  sub.Lang,
			}).Error("Failed to create subtitle node")
			continue
		}
	}
}
