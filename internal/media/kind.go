// Package media odpowiada za pliki na dysku: zapis, sumy kontrolne
// i generowanie miniatur. Nie zna ani HTTP, ani bazy — dostaje strumień
// i nazwę, zwraca metadane do zapisania.
package media

import (
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"designfolio/internal/model"
)

// extraTypes uzupełnia tablicę systemową o formaty, których Windows ani Linux
// zwykle nie znają — a są w tym katalogu pierwszoklasowymi obywatelami.
var extraTypes = map[string]string{
	".stl":   "model/stl",
	".3mf":   "model/3mf",
	".obj":   "model/obj",
	".step":  "model/step",
	".stp":   "model/step",
	".f3d":   "application/x-fusion360",
	".blend": "application/x-blender",
	".kra":   "application/x-krita",
	".psd":   "image/vnd.adobe.photoshop",
	".procreate": "application/x-procreate",
	".gcode": "text/x-gcode",
	".3ds":   "model/3ds",
	".webp":  "image/webp",
	".avif":  "image/avif",
	".md":    "text/markdown",
}

// DetectMime rozpoznaje typ pliku: najpierw po rozszerzeniu (pewniejsze dla
// formatów 3D, które są zwykłym tekstem albo zipem), potem przez sniffing
// pierwszych bajtów.
func DetectMime(filename string, head []byte) string {
	ext := strings.ToLower(filepath.Ext(filename))

	if t, ok := extraTypes[ext]; ok {
		return t
	}
	if t := mime.TypeByExtension(ext); t != "" {
		// TypeByExtension dokleja czasem "; charset=utf-8" — obcinamy,
		// bo w bazie trzymamy sam typ.
		if base, _, found := strings.Cut(t, ";"); found {
			return strings.TrimSpace(base)
		}
		return t
	}
	if len(head) > 0 {
		return http.DetectContentType(head)
	}
	return "application/octet-stream"
}

// KindForMime grupuje typ MIME w kategorię, która ma znaczenie dla interfejsu:
// obraz da się pokazać, model obejrzeć w 3D, resztę tylko pobrać.
func KindForMime(mimeType, filename string) model.AssetKind {
	ext := strings.ToLower(filepath.Ext(filename))

	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return model.AssetImage
	case strings.HasPrefix(mimeType, "model/"), ext == ".stl", ext == ".3mf", ext == ".obj":
		return model.AssetModel
	case strings.HasPrefix(mimeType, "text/"),
		mimeType == "application/pdf",
		mimeType == "application/json",
		strings.Contains(mimeType, "document"),
		strings.Contains(mimeType, "spreadsheet"):
		return model.AssetDocument
	case strings.Contains(mimeType, "zip"),
		strings.Contains(mimeType, "tar"),
		strings.Contains(mimeType, "rar"),
		strings.Contains(mimeType, "7z"),
		strings.Contains(mimeType, "compressed"):
		return model.AssetArchive
	default:
		return model.AssetOther
	}
}

// unsafeChars to wszystko, co nie jest literą, cyfrą, kropką, myślnikiem
// ani podkreślnikiem. Polskie znaki przechodzą — nazwa pliku ma zostać czytelna.
var unsafeChars = regexp.MustCompile(`[^\p{L}\p{N}._-]+`)

// MaxFilenameLen ogranicza nazwę, żeby ścieżka nie przekroczyła limitu systemu.
const MaxFilenameLen = 120

// SafeFilename czyści nazwę przysłaną przez przeglądarkę.
//
// To zabezpieczenie, a nie kosmetyka: bez niego nazwa "../../etc/passwd"
// albo "C:\Windows\system32\x" pozwoliłaby zapisać plik poza katalogiem danych.
// filepath.Base ucina katalogi, a podmiana znaków usuwa resztę niespodzianek.
func SafeFilename(name string) string {
	// Przeglądarki wysyłają czasem pełną ścieżkę z Windowsa — Base tego nie tnie,
	// bo na Linuksie backslash jest zwykłym znakiem. Robimy to ręcznie.
	name = strings.ReplaceAll(name, `\`, "/")
	name = filepath.Base(strings.TrimSpace(name))

	name = unsafeChars.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-.")

	if name == "" {
		name = "plik"
	}
	if len([]rune(name)) > MaxFilenameLen {
		ext := filepath.Ext(name)
		stem := strings.TrimSuffix(name, ext)
		keep := MaxFilenameLen - len([]rune(ext))
		if keep < 1 {
			keep = 1
		}
		name = string([]rune(stem)[:keep]) + ext
	}
	return name
}
