package controllers

import (
	"github.com/astaxie/beego"
	"time"
	"strings"
	"log"
	"net/http"
	"fmt"
	"io/ioutil"
	"encoding/json"
	"html"
	"html/template"
	"net/url"
)

const pornhubAPIURL = "http://www.pornhub.com/webmasters/"
const pornhubAPITimeout = 3
const pornhubCacheDuration = time.Minute * 5

// PornhubSearchResult type for pornhub api search result
type PornhubSearchResult map[string]interface{}

// PornhubSingleVideo type for pornhub api single video result
type PornhubSingleVideo map[string]interface{}

// PornhubEmbedCode type for pornhub api embed code
type PornhubEmbedCode map[string]interface{}

// PornhubController Beego Controller 
type PornhubController struct {
	beego.Controller
}

// Get Pornhub Video controller
func (c *PornhubController) Get() {
	// Get videoID from URL
	aux := strings.Replace(c.Ctx.Request.URL.Path, ".html", "", -1)
	str := strings.Split(aux, "/")
	videoID := str[2]

	// Build redirect URL in case the API fails
	redirect := "https://pornhub.com/view_video.php?viewkey=" + videoID + "&t=1&utm_source=just-tit.com&utm_medium=embed&utm_campaign=hubtraffic_dsmatilla"

	// Get base domain from URL
	BaseDomain := c.Controller.Ctx.Input.Scheme() + "://" + c.Controller.Ctx.Input.Domain()

	// Call the API and 307 redirect to fallback URL if something is not right
	data := PornhubGetVideoByID(videoID)
	_, ok := data["video"]
	if !ok { 
		c.Redirect(redirect, 307) 
		return
	}


	// Get Embed Code from API
	embedcode := PornhubGetVideoEmbedCode(videoID)
	if embedcode["embed"] == nil { 
		c.Redirect(redirect, 307) 
		return
	}
	embed := embedcode["embed"].(map[string]interface{})

	result := []JTVideo{}
	// Construct video object
	v := data["video"].(map[string]interface{})
	video := JTVideo{}
	video.ID = videoID
	video.Provider = "pornhub"
	video.Domain = BaseDomain
	video.Title = fmt.Sprintf("%s", v["title"])
	video.Description = fmt.Sprintf("%s", v["title"])
	video.Thumb = fmt.Sprintf("%s", v["thumb"])
	video.Embed = template.HTML(fmt.Sprintf("%+v", html.UnescapeString(embed["code"].(string))))
	video.URL = fmt.Sprintf(BaseDomain+"/pornhub/%s.html", videoID)
	video.Width = fmt.Sprintf("%s", v["width"])
	video.Height = fmt.Sprintf("%s", v["height"])
	video.Duration = fmt.Sprintf("%s", v["duration"])
	video.Views = fmt.Sprintf("%s", v["views"])
	video.Rating = fmt.Sprintf("%s", v["rating"])
	video.Ratings = fmt.Sprintf("%s", v["ratings"])
	video.Segment = fmt.Sprintf("%s", v["segment"])
	video.PublishDate = fmt.Sprintf("%s", v["publish_date"])
	for _, tags := range v["tags"].([]interface{}) {
		video.Tags = append(video.Tags, fmt.Sprintf("%s", tags.(map[string]interface {})["tag_name"]))
	}
	for _, categories := range v["categories"].([]interface{}) {
		video.Categories = append(video.Categories, fmt.Sprintf("%s", categories.(map[string]interface {})["category"]))
	}
	for _, thumbs := range v["thumbs"].([]interface{}) {
		video.Thumbs = append(video.Thumbs, fmt.Sprintf("%s", thumbs.(map[string]interface {})["src"]))
	}
	for _, pornstars := range v["pornstars"].([]interface{}) {
		video.Pornstars = append(video.Pornstars, fmt.Sprintf("%s", pornstars.(map[string]interface {})["pornstar_name"]))
	}
	video.ExternalID = fmt.Sprintf("%s", v["video_id"])
	video.ExternalURL = fmt.Sprintf("%s", v["url"])

	result = append(result, video)

	// Send object to template
	c.Data["Result"] = result
	c.Layout = "index.tpl"
	c.TplName = "singlevideo.tpl"
}

// PornhubGetVideoByID retrieves specific video data from Pornhub API
func PornhubGetVideoByID(ID string) PornhubSingleVideo {
	Cached := JTCache.Get("pornhub-video-"+ID)
	var result PornhubSingleVideo
	if Cached == nil {
		timeout := time.Duration(pornhubAPITimeout * time.Second)
		client := http.Client{
			Timeout: timeout,
		}
		resp, err := client.Get(fmt.Sprintf(pornhubAPIURL+"video_by_id?id=%s", ID))
		if err != nil {
			log.Println("[PORNHUB][GETVIDEOBYID]", err)
			return PornhubSingleVideo{}
		}
		b, _ := ioutil.ReadAll(resp.Body)

		err = json.Unmarshal(b, &result)
		if err != nil {
			log.Println("[PORNHUB][GETVIDEOBYID]", err)
		}
		JTCache.Put("pornhub-video-"+ID, b, pornhubCacheDuration)
	} else {
		json.Unmarshal(Cached.([]uint8), &result)
	}
	return result
}

// PornhubGetVideoEmbedCode retrieves specific video embed code from Pornhub API
func PornhubGetVideoEmbedCode(ID string) PornhubEmbedCode {
	Cached := JTCache.Get("pornhub-embed-"+ID)
	if Cached == nil {
		timeout := time.Duration(pornhubAPITimeout * time.Second)
		client := http.Client{
			Timeout: timeout,
		}
		resp, err := client.Get(fmt.Sprintf(pornhubAPIURL+"video_embed_code?id=%s", ID))
		if err != nil {
			log.Println("[PORNHUB][GETVIDEOEMBEDCODE]",err)
			return PornhubEmbedCode{}
		}
		b, _ := ioutil.ReadAll(resp.Body)
		var result PornhubEmbedCode
		err = json.Unmarshal(b, &result)
		if err != nil {
			log.Println("[PORNHUB][GETVIDEOEMBEDCODE]",err)
		}
		JTCache.Put("pornhub-embed-"+ID, b, pornhubCacheDuration)
		return result
	}
	var result PornhubEmbedCode
	json.Unmarshal(Cached.([]uint8), &result)
	return result
}

// PornhubSearch Calls porhub search function and process result to get array of videos
func PornhubSearch(search string) []JTVideo {
	videos := PornhubSearchVideos(search)
	result := []JTVideo{}
	for _, data := range videos["videos"].([]interface{}){
		// Construct video object
		v := data.(interface{})
		video := JTVideo{}
		video.ID = fmt.Sprintf("%s", v.(map[string]interface {})["video_id"])
		video.Provider = "pornhub"
		video.Title = fmt.Sprintf("%s",  v.(map[string]interface {})["title"])
		video.Description = fmt.Sprintf("%s",  v.(map[string]interface {})["title"])
		video.Thumb = fmt.Sprintf("%s",  v.(map[string]interface {})["thumb"])
		video.Width = fmt.Sprintf("%s", v.(map[string]interface {})["width"])
		video.Height = fmt.Sprintf("%s",  v.(map[string]interface {})["height"])
		video.Duration = fmt.Sprintf("%s", v.(map[string]interface {})["duration"])
		video.Views = fmt.Sprintf("%s",  v.(map[string]interface {})["views"])
		video.Rating = fmt.Sprintf("%s",  v.(map[string]interface {})["rating"])
		video.Ratings = fmt.Sprintf("%s",  v.(map[string]interface {})["ratings"])
		video.Segment = fmt.Sprintf("%s",  v.(map[string]interface {})["segment"])
		video.PublishDate = fmt.Sprintf("%s", v.(map[string]interface {})["publish_date"])
		video.ExternalID = fmt.Sprintf("%s", v.(map[string]interface {})["video_id"])
		video.ExternalURL = fmt.Sprintf("%s", v.(map[string]interface {})["url"])

		tags := v.(map[string]interface {})["tags"]
		for _, tag := range tags.([]interface{}) {
			video.Tags = append(video.Tags, fmt.Sprintf("%s", tag.(map[string]interface {})["tag_name"]))
		}
		categories := v.(map[string]interface {})["categories"]
		for _, category := range categories.([]interface{}) {
			video.Categories = append(video.Categories, fmt.Sprintf("%s", category.(map[string]interface {})["category"]))
		}
		thumbs := v.(map[string]interface {})["thumbs"]
		for _, thumb := range thumbs.([]interface{}) {
			video.Thumbs = append(video.Thumbs, fmt.Sprintf("%s", thumb.(map[string]interface {})["src"]))
		}
		pornstars := v.(map[string]interface {})["pornstars"]
		for _, pornstar := range pornstars.([]interface{}) {
			video.Pornstars = append(video.Pornstars, fmt.Sprintf("%s", pornstar.(map[string]interface {})["pornstar_name"]))
		}
		result = append(result, video)
	}
	return result
}

// PornhubSearchVideos Search string in pornhub search API
func PornhubSearchVideos(search string) PornhubSearchResult {
	Cached := JTCache.Get("pornhub-search-"+search)
	if Cached == nil {
		timeout := time.Duration(pornhubAPITimeout * time.Second)
		client := http.Client{
			Timeout: timeout,
		}
		resp, err := client.Get(fmt.Sprintf(pornhubAPIURL+"search?search=%s&thumbnail=all", url.QueryEscape(search)))
		if err != nil {
			log.Println("[PORNHUB][SEARCHVIDEOS]",err)
			return PornhubSearchResult{}
		}
		b, _ := ioutil.ReadAll(resp.Body)
		var result PornhubSearchResult
		err = json.Unmarshal(b, &result)
		if err != nil {
			log.Println("[PORNHUB][SEARCHVIDEOS]",err)
		}
		JTCache.Put("pornhub-search-"+search, b, pornhubCacheDuration)
		return result
	}
	var result PornhubSearchResult
	json.Unmarshal(Cached.([]uint8), &result)
	return result
}