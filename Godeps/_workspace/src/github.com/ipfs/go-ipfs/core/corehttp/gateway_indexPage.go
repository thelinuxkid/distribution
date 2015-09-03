package corehttp

import (
	"github.com/ipfs/go-ipfs/assets"
	"html/template"
	"path"
	"strings"
)

// structs for directory listing
type listingTemplateData struct {
	Listing  []directoryItem
	Path     string
	BackLink string
}

type directoryItem struct {
	Size uint64
	Name string
	Path string
}

var listingTemplate *template.Template

func init() {
	assetPath := "../vendor/src/QmeNXKecZ7CQagtkQUJxG3yS7UcvU6puS777dQsx3amkS7/dir-index-html/"
	knownIconsBytes, err := assets.Asset(assetPath + "knownIcons.txt")
	if err != nil {
		panic(err)
	}
	knownIcons := make(map[string]struct{})
	for _, ext := range strings.Split(strings.TrimSuffix(string(knownIconsBytes), "\n"), "\n") {
		knownIcons[ext] = struct{}{}
	}

	// helper to guess the type/icon for it by the extension name
	iconFromExt := func(name string) string {
		ext := path.Ext(name)
		_, ok := knownIcons[ext]
		if !ok {
			// default blank icon
			return "ipfs-_blank"
		}
		return "ipfs-" + ext[1:] // slice of the first dot
	}

	// Directory listing template
	dirIndexBytes, err := assets.Asset(assetPath + "dir-index.html")
	if err != nil {
		panic(err)
	}

	listingTemplate = template.Must(template.New("dir").Funcs(template.FuncMap{
		"iconFromExt": iconFromExt,
	}).Parse(string(dirIndexBytes)))
}
