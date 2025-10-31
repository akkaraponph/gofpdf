package gofpdf

import (
	"embed"
	"io"
	"io/fs"
	"path"
	"strings"
)

//go:embed font/th/*/*.ttf
var embeddedFonts embed.FS

// EmbeddedFontLoader implements FontLoader interface to load fonts from embedded FS
type EmbeddedFontLoader struct {
	fs embed.FS
}

// Open implements FontLoader.Open for embedded fonts
func (e *EmbeddedFontLoader) Open(name string) (io.Reader, error) {
	return e.fs.Open(name)
}

// NewEmbeddedFontLoader creates a FontLoader that reads from the embedded font directory
func NewEmbeddedFontLoader() *EmbeddedFontLoader {
	return &EmbeddedFontLoader{fs: embeddedFonts}
}

// UseEmbeddedFonts configures the Fpdf instance to use embedded fonts from the package.
// After calling this, you can use SetFont with any bundled Thai font family without
// needing to call SetFontLocation or worry about file paths.
//
// Example:
//
//	pdf := gofpdf.New("P", "mm", "A4", "")
//	pdf.UseEmbeddedFonts()
//	pdf.AddPage()
//	pdf.SetFont("Kanit", "", 14)  // Works automatically with embedded fonts
func (f *Fpdf) UseEmbeddedFonts() {
	f.SetFontLoader(NewEmbeddedFontLoader())
	f.fontpath = "font/th" // Default to Thai fonts, but subfolder search will work
}

// embeddedFontReader wraps embed.FS to implement the file reading for AddUTF8FontFromBytes
type embeddedFontReader struct {
	basePath string
}

// tryLoadEmbeddedFont attempts to load a font from the embedded filesystem
func (f *Fpdf) tryLoadEmbeddedFont(familyStr, styleStr string) bool {
	if f.fontLoader == nil {
		return false
	}

	// Try common patterns for embedded fonts
	var styleSuffixes []string
	switch styleStr {
	case "B":
		styleSuffixes = []string{"-Bold", "Bold", "Bd", "B"}
	case "I":
		styleSuffixes = []string{"-Italic", "Italic", "It", "I", "-Oblique", "Oblique"}
	case "BI":
		styleSuffixes = []string{"-BoldItalic", "BoldItalic", "BI", "-Bold-Italic", "Bold-Italic"}
	default:
		styleSuffixes = []string{"", "-Regular", "Regular"}
	}

	// FIX: Try both lowercase and capitalized versions of family name
	// Since SetFont() converts to lowercase but folders are capitalized
	familyVariations := []string{
		familyStr, // lowercase (e.g., "tahoma")
		strings.Title(strings.ToLower(familyStr)), // Capitalized (e.g., "Tahoma")
	}

	// Search paths: font/th/FamilyName/ (try both cases)
	for _, familyVar := range familyVariations {
		searchPath := path.Join("font/th", familyVar)
		
		for _, suf := range styleSuffixes {
			for _, ext := range []string{".ttf", ".otf"} {
				var filename string
				if suf == "" {
					filename = familyVar + ext  // ← FIX: Use familyVar (capitalized) not familyStr
				} else {
					filename = familyVar + suf + ext  // ← FIX: Use familyVar
				}
				
				fullPath := path.Join(searchPath, filename)

				// Try to open the file from embedded FS
				file, err := f.fontLoader.Open(fullPath)
				if err == nil {
					// Close if it implements io.Closer
					if closer, ok := file.(io.Closer); ok {
						closer.Close()
					}
					// File exists, use AddUTF8Font with the path
					f.AddUTF8Font(familyStr, styleStr, fullPath)
					if f.err == nil {
						if _, ok := f.fonts[getFontKey(familyStr, styleStr)]; ok {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

// GetEmbeddedFontFamilies returns a list of all available font families in the embedded fonts
func GetEmbeddedFontFamilies() []string {
	families := make(map[string]bool)

	// Walk through embedded font directories
	fs.WalkDir(embeddedFonts, "font/th", func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		// Extract family name from path like "font/th/Kanit"
		parts := strings.Split(path, "/")
		if len(parts) == 3 {
			families[parts[2]] = true
		}
		return nil
	})

	// Convert map to sorted slice
	result := make([]string, 0, len(families))
	for family := range families {
		result = append(result, family)
	}

	return result
}
