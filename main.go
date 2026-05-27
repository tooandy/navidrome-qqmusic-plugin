// QQMusic plugin for Navidrome - fetches artist and album artwork from QQ Music.
//
// Build with:
//
//	tinygo build -o qqmusic.wasm -target wasip1 -buildmode=c-shared .
//
// Install by copying the .ndp file to your Navidrome plugins folder.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/metadata"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

// CoverSize constants matching QQMusic's size segments
const (
	CoverSize150  = 150
	CoverSize300  = 300
	CoverSize500  = 500
	CoverSize800  = 800
)

var sizeSegments = map[int]string{
	CoverSize150:  "R150x150",
	CoverSize300:  "R300x300",
	CoverSize500:  "R500x500",
	CoverSize800:  "R800x800",
}

// PhotoNewURLKind constants
const (
	PhotoNewKindArtist = "T001"
	PhotoNewKindAlbum  = "T002"
)

// qqmusicPlugin implements the metadata provider interfaces for artwork.
type qqmusicPlugin struct{}

// init registers the plugin implementation
func init() {
	metadata.Register(&qqmusicPlugin{})
}

// Ensure qqmusicPlugin implements the provider interfaces
var (
	_ metadata.ArtistImagesProvider = (*qqmusicPlugin)(nil)
	_ metadata.AlbumImagesProvider  = (*qqmusicPlugin)(nil)
)

// buildCoverURL constructs cover URL from mid and kind
func buildCoverURL(kind string, mid string, size int) string {
	seg, ok := sizeSegments[size]
	if !ok {
		seg = sizeSegments[CoverSize300]
	}
	return "https://y.gtimg.cn/music/photo_new/" + kind + seg + "M000" + mid + ".jpg"
}

// extractMidFromPicURL extracts mid from pic URL
func extractMidFromPicURL(picURL string) string {
	if picURL == "" {
		return ""
	}
	for _, kind := range []string{PhotoNewKindArtist, PhotoNewKindAlbum} {
		for _, seg := range sizeSegments {
			prefix := kind + seg + "M000"
			if idx := strings.Index(picURL, prefix); idx >= 0 {
				start := idx + len(prefix)
				end := strings.Index(picURL[start:], ".jpg")
				if end > 0 {
					return picURL[start : start+end]
				}
			}
		}
	}
	if idx := strings.Index(picURL, "/photo/mid/"); idx >= 0 {
		start := idx + len("/photo/mid/")
		end := strings.Index(picURL[start:], ".jpg")
		if end > 0 {
			return picURL[start : start+end]
		}
	}
	return ""
}

// SearchType values
const (
	SearchTypeSong   = 0
	SearchTypeSinger = 1
	SearchTypeAlbum  = 2
)

// SearchResult represents QQ Music search response
type SearchResult struct {
	Code int `json:"code"`
	Data struct {
		Singer []SingerSearch `json:"singer"`
		Album  []AlbumSearch  `json:"album"`
	} `json:"data"`
}

type SingerSearch struct {
	ID   int    `json:"id"`
	Mid  string `json:"mid"`
	Name string `json:"name"`
	Pmid string `json:"pmid"`
	Pic  string `json:"pic"`
}

type AlbumSearch struct {
	ID    int    `json:"id"`
	Mid   string `json:"mid"`
	Name  string `json:"name"`
	Title string `json:"title"`
	Pic   string `json:"pic"`
}

// searchQQMusic searches QQ Music by keyword and type
func searchQQMusic(keyword string, searchType int) (*SearchResult, error) {
	// Build URL with query params
	url := "https://c.y.qq.com/splcloud/fcgi-bin/smartbox_new.fcg?key=" + keyword + "&search_type=" + fmt.Sprintf("%d", searchType) + "&num_per_page=10&page_num=1"

	resp, err := host.HTTPSend(host.HTTPRequest{
		Method: "GET",
		URL:    url,
		Headers: map[string]string{
			"User-Agent": "NavidromeQQMusicPlugin/1.0",
		},
		TimeoutMs: 10000,
	})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search failed with status: %d", resp.StatusCode)
	}

	var result SearchResult
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAlbumImages returns album cover images from QQ Music
func (*qqmusicPlugin) GetAlbumImages(input metadata.AlbumRequest) (*metadata.AlbumImagesResponse, error) {
	pdk.Log(pdk.LogDebug, "GetAlbumImages: name="+input.Name+", artist="+input.Artist)

	result, err := searchQQMusic(input.Name, SearchTypeAlbum)
	if err != nil {
		pdk.Log(pdk.LogDebug, "QQ Music search failed: "+err.Error())
		return nil, err
	}

	// Find matching album
	var bestMatch *AlbumSearch
	for i := range result.Data.Album {
		album := &result.Data.Album[i]
		if strings.EqualFold(album.Name, input.Name) {
			if input.Artist != "" {
				// TODO: check singer_list for artist match
				bestMatch = album
				break
			} else {
				bestMatch = album
				break
			}
		}
	}

	// Fallback to first result
	if bestMatch == nil && len(result.Data.Album) > 0 {
		bestMatch = &result.Data.Album[0]
	}

	if bestMatch == nil || (bestMatch.Mid == "" && bestMatch.Pic == "") {
		pdk.Log(pdk.LogInfo, "Album not found in QQ Music: "+input.Name)
		return nil, errors.New("album not found")
	}

	mid := bestMatch.Mid
	if mid == "" && bestMatch.Pic != "" {
		mid = extractMidFromPicURL(bestMatch.Pic)
	}

	if mid == "" {
		pdk.Log(pdk.LogInfo, "Could not extract album mid for: "+input.Name)
		return nil, errors.New("album mid not found")
	}

	images := []metadata.ImageInfo{
		{URL: buildCoverURL(PhotoNewKindAlbum, mid, CoverSize150), Size: 150},
		{URL: buildCoverURL(PhotoNewKindAlbum, mid, CoverSize300), Size: 300},
		{URL: buildCoverURL(PhotoNewKindAlbum, mid, CoverSize500), Size: 500},
		{URL: buildCoverURL(PhotoNewKindAlbum, mid, CoverSize800), Size: 800},
	}

	pdk.Log(pdk.LogDebug, "Found album images for: "+input.Name)
	return &metadata.AlbumImagesResponse{Images: images}, nil
}

// GetArtistImages returns artist images from QQ Music
func (*qqmusicPlugin) GetArtistImages(input metadata.ArtistRequest) (*metadata.ArtistImagesResponse, error) {
	pdk.Log(pdk.LogDebug, "GetArtistImages: name="+input.Name+", mbid="+input.MBID)

	result, err := searchQQMusic(input.Name, SearchTypeSinger)
	if err != nil {
		pdk.Log(pdk.LogDebug, "QQ Music search failed: "+err.Error())
		return nil, err
	}

	// Find matching singer
	var bestMatch *SingerSearch
	for i := range result.Data.Singer {
		singer := &result.Data.Singer[i]
		if strings.EqualFold(singer.Name, input.Name) {
			bestMatch = singer
			break
		}
	}

	// Fallback to first result
	if bestMatch == nil && len(result.Data.Singer) > 0 {
		bestMatch = &result.Data.Singer[0]
	}

	if bestMatch == nil || (bestMatch.Mid == "" && bestMatch.Pic == "") {
		pdk.Log(pdk.LogInfo, "Artist not found in QQ Music: "+input.Name)
		return nil, errors.New("artist not found")
	}

	mid := bestMatch.Mid
	if mid == "" && bestMatch.Pic != "" {
		mid = extractMidFromPicURL(bestMatch.Pic)
	}

	if mid == "" {
		pdk.Log(pdk.LogInfo, "Could not extract artist mid for: "+input.Name)
		return nil, errors.New("artist mid not found")
	}

	images := []metadata.ImageInfo{
		{URL: buildCoverURL(PhotoNewKindArtist, mid, CoverSize150), Size: 150},
		{URL: buildCoverURL(PhotoNewKindArtist, mid, CoverSize300), Size: 300},
		{URL: buildCoverURL(PhotoNewKindArtist, mid, CoverSize500), Size: 500},
		{URL: buildCoverURL(PhotoNewKindArtist, mid, CoverSize800), Size: 800},
	}

	pdk.Log(pdk.LogDebug, "Found artist images for: "+input.Name)
	return &metadata.ArtistImagesResponse{Images: images}, nil
}

// Required main function - init() handles registration
func main() {}