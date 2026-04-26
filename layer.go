package ldtkgo

import (
	"image"
	"image/png"
	"io/fs"
	"path"
)

// LayerType represents the type of a layer image from the simplified export.
type LayerType int

const (
	LayerBackground LayerType = iota // _bg.png
	LayerTiles                       // Named tile layer (e.g., tiles.png)
	LayerComposite                   // _composite.png
)

// Layer represents a loaded layer image from the simplified export.
type Layer struct {
	Name  string
	Type  LayerType
	Image image.Image
}

// LoadLayers loads the layer images for this level from the simplified export directory.
// dir is the path to the level folder (e.g., "pong/simplified/level_0").
// Background (_bg.png) and composite (_composite.png) are included when present.
// Returns layers in order: background, tile layers (from data.json), composite.
func (level *Level) LoadLayers(dir string, fileSystem fs.FS) ([]*Layer, error) {
	var layers []*Layer

	// Background layer (_bg.png) — optional
	bgPath := path.Join(dir, "_bg.png")
	if bgImg, err := loadPNG(bgPath, fileSystem); err == nil {
		layers = append(layers, &Layer{Name: "_bg.png", Type: LayerBackground, Image: bgImg})
	}

	// Tile layers listed in data.json
	for _, name := range level.Layers {
		tilePath := path.Join(dir, name)
		img, err := loadPNG(tilePath, fileSystem)
		if err != nil {
			return nil, err
		}
		layers = append(layers, &Layer{Name: name, Type: LayerTiles, Image: img})
	}

	// Composite layer (_composite.png) — optional
	compPath := path.Join(dir, "_composite.png")
	if compImg, err := loadPNG(compPath, fileSystem); err == nil {
		layers = append(layers, &Layer{Name: "_composite.png", Type: LayerComposite, Image: compImg})
	}

	return layers, nil
}

// LoadBGImage loads the background image referenced in the .ldtk project file.
// basePath is the directory containing the .ldtk file. Requires prior LinkProject call.
func (level *Level) LoadBGImage(basePath string, fileSystem fs.FS) error {
	if level.BGRelPath == "" {
		return nil
	}
	filePath := path.Join(basePath, level.BGRelPath)
	img, err := loadPNG(filePath, fileSystem)
	if err != nil {
		return err
	}
	level.BGImage = img
	return nil
}

// loadPNG loads a PNG image from the file system.
func loadPNG(filePath string, fileSystem fs.FS) (image.Image, error) {
	file, err := fileSystem.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}
