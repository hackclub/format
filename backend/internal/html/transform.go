package html

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/hackclub/format/internal/assets"
)

type Transformer struct {
	assetService *assets.Service
}

type TransformRequest struct {
	HTML string `json:"html"`
}

type TransformResponse struct {
	HTML     string   `json:"html"`
	Messages []string `json:"messages,omitempty"`
	Stats    Stats    `json:"stats"`
}

type Stats struct {
	ImagesProcessed int `json:"images_processed"`
	ImagesRehosted  int `json:"images_rehosted"`
	StylesRemoved   int `json:"styles_removed"`
	ScriptsRemoved  int `json:"scripts_removed"`
}

func NewTransformer(assetService *assets.Service) *Transformer {
	return &Transformer{
		assetService: assetService,
	}
}

// Transform processes HTML and rehoists images, sanitizes content
func (t *Transformer) Transform(ctx context.Context, req *TransformRequest) (*TransformResponse, error) {
	html := req.HTML
	stats := Stats{}
	messages := []string{}

	// 1. Extract and process images
	html, imageStats, imageMessages := t.processImages(ctx, html)
	stats.ImagesProcessed = imageStats.ImagesProcessed
	stats.ImagesRehosted = imageStats.ImagesRehosted
	messages = append(messages, imageMessages...)

	// 2. Sanitize HTML
	html, sanitizeStats := t.sanitizeHTML(html)
	stats.StylesRemoved = sanitizeStats.StylesRemoved
	stats.ScriptsRemoved = sanitizeStats.ScriptsRemoved

	return &TransformResponse{
		HTML:     html,
		Messages: messages,
		Stats:    stats,
	}, nil
}

// processImages finds all img tags and rehoists external/data images
func (t *Transformer) processImages(ctx context.Context, html string) (string, Stats, []string) {
	stats := Stats{}
	messages := []string{}

	// Regex to find img tags
	imgRegex := regexp.MustCompile(`<img[^>]*src=["']([^"']+)["'][^>]*>`)
	srcRegex := regexp.MustCompile(`src=["']([^"']+)["']`)

	matches := imgRegex.FindAllStringSubmatch(html, -1)
	stats.ImagesProcessed = len(matches)

	// Process each image
	for _, match := range matches {
		fullImgTag := match[0]
		srcURL := match[1]

		// Skip if already using our CDN
		if strings.Contains(srcURL, "i.format.hackclub.com") {
			continue
		}

		// Handle blob URLs (Gmail draft images)
		if strings.HasPrefix(srcURL, "blob:") {
			messages = append(messages, "Gmail draft images detected - please download and re-upload images manually for rehosting")
			continue
		}

		// Handle Gmail attachment URLs (require authentication)
		if strings.Contains(srcURL, "mail.google.com") && strings.Contains(srcURL, "attid=") {
			messages = append(messages, "Gmail attachment image detected - please download and re-upload manually for rehosting")
			continue
		}

		// Check if we should rehost this image
		shouldRehost := t.shouldRehostImage(srcURL)
		if !shouldRehost {
			continue
		}

		// Process the image
		var asset *assets.Asset
		var err error

		if strings.HasPrefix(srcURL, "data:") {
			asset, err = t.assetService.ProcessFromDataURI(ctx, srcURL)
		} else {
			asset, err = t.assetService.ProcessFromURL(ctx, srcURL)
		}

		if err != nil {
			messages = append(messages, fmt.Sprintf("Failed to rehost image %s: %v", srcURL[:min(50, len(srcURL))], err))
			continue
		}

		messages = append(messages, fmt.Sprintf("Image rehosted: %s -> %s", srcURL[:min(50, len(srcURL))], asset.URL))

		// Replace the src in the img tag
		newImgTag := srcRegex.ReplaceAllString(fullImgTag, fmt.Sprintf(`src="%s"`, asset.URL))
		
		// Add alt text if missing
		if !strings.Contains(newImgTag, "alt=") {
			newImgTag = strings.Replace(newImgTag, ">", ` alt="">`, 1)
		}

		// Add Gmail-safe styling
		newImgTag = t.addGmailSafeImageStyles(newImgTag)

		html = strings.Replace(html, fullImgTag, newImgTag, 1)
		stats.ImagesRehosted++

		if asset.Deduped {
			messages = append(messages, fmt.Sprintf("Image deduplicated: %s", asset.URL))
		} else {
			messages = append(messages, fmt.Sprintf("Image rehosted: %s", asset.URL))
		}
	}

	return html, stats, messages
}

// shouldRehostImage determines if an image should be rehosted
func (t *Transformer) shouldRehostImage(srcURL string) bool {
	// Always rehost data URIs
	if strings.HasPrefix(srcURL, "data:") {
		return true
	}

	// Cannot rehost blob URLs (browser-generated temporary URLs)
	if strings.HasPrefix(srcURL, "blob:") {
		return false
	}

	// Parse URL
	parsedURL, err := url.Parse(srcURL)
	if err != nil {
		return false
	}

	// Rehost if not HTTPS
	if parsedURL.Scheme != "https" {
		return true
	}

	// Rehost common temporary/signed URL patterns
	host := parsedURL.Host
	if strings.Contains(host, "amazonaws.com") ||
		strings.Contains(host, "googleusercontent.com") ||
		strings.Contains(host, "mail.google.com") ||
		strings.Contains(host, "notion.so") ||
		strings.Contains(host, "dropbox.com") ||
		strings.Contains(host, "onedrive.com") {
		return true
	}

	// Check for signed URL patterns
	if strings.Contains(parsedURL.RawQuery, "Expires=") ||
		strings.Contains(parsedURL.RawQuery, "expires=") ||
		strings.Contains(parsedURL.RawQuery, "X-Amz-") ||
		strings.Contains(parsedURL.RawQuery, "sig=") ||
		strings.Contains(parsedURL.RawQuery, "token=") {
		return true
	}

	return false
}

// addGmailSafeImageStyles adds Gmail-compatible styling to img tags
func (t *Transformer) addGmailSafeImageStyles(imgTag string) string {
	style := `style="max-width:100%;height:auto;display:block;"`
	
	if strings.Contains(imgTag, "style=") {
		// Replace existing style attribute
		styleRegex := regexp.MustCompile(`style=["'][^"']*["']`)
		imgTag = styleRegex.ReplaceAllString(imgTag, style)
	} else {
		// Add style attribute
		imgTag = strings.Replace(imgTag, ">", " "+style+">", 1)
	}
	
	return imgTag
}

// sanitizeHTML removes dangerous elements and converts everything to Gmail format
func (t *Transformer) sanitizeHTML(html string) (string, Stats) {
	stats := Stats{}

	// Remove script tags
	scriptRegex := regexp.MustCompile(`<script[^>]*>.*?</script>`)
	scripts := scriptRegex.FindAllString(html, -1)
	html = scriptRegex.ReplaceAllString(html, "")
	stats.ScriptsRemoved = len(scripts)

	// Remove style tags (but not inline styles)
	styleTagRegex := regexp.MustCompile(`<style[^>]*>.*?</style>`)
	styleTags := styleTagRegex.FindAllString(html, -1)
	html = styleTagRegex.ReplaceAllString(html, "")
	stats.StylesRemoved = len(styleTags)

	// Always convert to Gmail-compatible format
	html = t.convertToGmailFormat(html)

	// Remove dangerous attributes
	html = t.removeDangerousAttributes(html)

	// Normalize links (including mailto: detection)
	html = t.normalizeLinks(html)

	return html, stats
}

// removeDangerousAttributes removes potentially dangerous HTML attributes
func (t *Transformer) removeDangerousAttributes(html string) string {
	// Remove onclick and other event handlers
	eventRegex := regexp.MustCompile(`\s+on\w+="[^"]*"`)
	html = eventRegex.ReplaceAllString(html, "")

	// Remove javascript: links
	jsLinkRegex := regexp.MustCompile(`href="javascript:[^"]*"`)
	html = jsLinkRegex.ReplaceAllString(html, `href="#"`)

	// Remove classes except gmail_quote (preserve Gmail-specific classes)
	classRegex := regexp.MustCompile(`\s+class="([^"]*)"`)
	html = classRegex.ReplaceAllStringFunc(html, func(match string) string {
		if strings.Contains(match, `class="gmail_quote"`) || strings.Contains(match, `class="gmail_`) {
			return match // Keep Gmail classes
		}
		return "" // Remove other classes
	})
	
	// Remove IDs (but be more careful)
	idRegex := regexp.MustCompile(`\s+id="[^"]*"`)
	html = idRegex.ReplaceAllString(html, "")

	return html
}

// normalizeLinks ensures all links are HTTPS and removes tracking
func (t *Transformer) normalizeLinks(html string) string {
	linkRegex := regexp.MustCompile(`<a[^>]*href="([^"]+)"[^>]*>`)
	
	return linkRegex.ReplaceAllStringFunc(html, func(match string) string {
		hrefRegex := regexp.MustCompile(`href="([^"]+)"`)
		hrefMatch := hrefRegex.FindStringSubmatch(match)
		if len(hrefMatch) != 2 {
			return match
		}
		
		originalURL := hrefMatch[1]
		cleanURL := t.cleanURL(originalURL)
		
		return strings.Replace(match, fmt.Sprintf(`href="%s"`, originalURL), fmt.Sprintf(`href="%s"`, cleanURL), 1)
	})
}

// cleanURL removes tracking parameters, ensures HTTPS, and detects email addresses
func (t *Transformer) cleanURL(urlStr string) string {
	// Check if it looks like an email address without mailto:
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if emailRegex.MatchString(urlStr) {
		return "mailto:" + urlStr
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}

	// If it's already a mailto: link, keep it as-is
	if parsedURL.Scheme == "mailto" {
		return urlStr
	}

	// Force HTTPS for http links
	if parsedURL.Scheme == "http" {
		parsedURL.Scheme = "https"
	}

	// Remove common tracking parameters
	trackingParams := []string{"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content", "gclid", "fbclid"}
	query := parsedURL.Query()
	
	for _, param := range trackingParams {
		query.Del(param)
	}
	
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}



// convertToGmailFormat converts ALL HTML to Gmail-compatible structure
func (t *Transformer) convertToGmailFormat(html string) string {
	// Base Gmail paragraph style
	const gmailParagraphStyle = `style="color: rgb(34, 34, 34); font-family: Arial, Helvetica, sans-serif; font-size: small; font-style: normal; font-variant-ligatures: normal; font-variant-caps: normal; font-weight: 400; letter-spacing: normal; orphans: 2; text-align: start; text-indent: 0px; text-transform: none; widows: 2; word-spacing: 0px; -webkit-text-stroke-width: 0px; white-space: normal; text-decoration-thickness: initial; text-decoration-style: initial; text-decoration-color: initial;"`

	// Convert paragraphs to Gmail format
	paragraphRegex := regexp.MustCompile(`<p[^>]*>(.*?)</p>`)
	html = paragraphRegex.ReplaceAllStringFunc(html, func(match string) string {
		// Extract content
		contentRegex := regexp.MustCompile(`<p[^>]*>(.*?)</p>`)
		matches := contentRegex.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		content := matches[1]
		
		// If content is just <br>, create a blank line div
		if content == "<br>" || content == "<br/>" || content == "<br />" {
			return `<div ` + gmailParagraphStyle + `><br></div>`
		}
		
		// Regular content div
		return `<div ` + gmailParagraphStyle + `>` + content + `</div>`
	})

	// Convert divs to Gmail format (normalize existing Gmail content)
	divRegex := regexp.MustCompile(`<div[^>]*>(.*?)</div>`)
	html = divRegex.ReplaceAllStringFunc(html, func(match string) string {
		// Skip if it's already a Gmail-style div or contains lists/blockquotes
		if strings.Contains(match, `color: rgb(34, 34, 34)`) || 
		   strings.Contains(match, `<ol>`) || strings.Contains(match, `<ul>`) || 
		   strings.Contains(match, `<blockquote`) {
			return match
		}
		
		// Extract content
		contentRegex := regexp.MustCompile(`<div[^>]*>(.*?)</div>`)
		matches := contentRegex.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		content := matches[1]
		
		// Create Gmail div
		return `<div ` + gmailParagraphStyle + `>` + content + `</div>`
	})

	// Convert headings to Gmail-style divs
	html = t.convertHeadingsToGmail(html)

	// Convert blockquotes to Gmail format
	blockquoteRegex := regexp.MustCompile(`<blockquote[^>]*>(.*?)</blockquote>`)
	html = blockquoteRegex.ReplaceAllString(html, 
		`<blockquote class="gmail_quote" style="color: rgb(34, 34, 34); font-family: Arial, Helvetica, sans-serif; font-size: small; font-style: normal; font-variant-ligatures: normal; font-variant-caps: normal; font-weight: 400; letter-spacing: normal; orphans: 2; text-align: start; text-indent: 0px; text-transform: none; widows: 2; word-spacing: 0px; -webkit-text-stroke-width: 0px; white-space: normal; text-decoration-thickness: initial; text-decoration-style: initial; text-decoration-color: initial; margin: 0px 0px 0px 0.8ex; border-left: 1px solid rgb(204, 204, 204); padding-left: 1ex;">$1</blockquote>`)

	// Ensure proper link styling
	linkRegex := regexp.MustCompile(`<a([^>]*?)>`)
	html = linkRegex.ReplaceAllStringFunc(html, func(match string) string {
		if !strings.Contains(match, "style=") {
			return strings.Replace(match, ">", ` style="color: rgb(17, 85, 204);">`, 1)
		}
		return match
	})

	return html
}



// convertHeadingsToGmail converts headings to Gmail-compatible divs
func (t *Transformer) convertHeadingsToGmail(html string) string {
	const gmailParagraphStyle = `style="color: rgb(34, 34, 34); font-family: Arial, Helvetica, sans-serif; font-style: normal; font-variant-ligatures: normal; font-variant-caps: normal; letter-spacing: normal; orphans: 2; text-align: start; text-indent: 0px; text-transform: none; widows: 2; word-spacing: 0px; -webkit-text-stroke-width: 0px; white-space: normal; text-decoration-thickness: initial; text-decoration-style: initial; text-decoration-color: initial;"`

	headingRegex := regexp.MustCompile(`<(h[1-6])[^>]*>(.*?)</h[1-6]>`)
	
	return headingRegex.ReplaceAllStringFunc(html, func(match string) string {
		submatches := headingRegex.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}
		
		level := submatches[1]
		content := submatches[2]
		
		// Gmail heading styles
		var fontSize, fontWeight string
		switch level {
		case "h1":
			fontSize = "font-size: large;"
			fontWeight = "font-weight: bold;"
		case "h2":
			fontSize = "font-size: medium;"
			fontWeight = "font-weight: bold;"
		case "h3", "h4", "h5", "h6":
			fontSize = "font-size: small;"
			fontWeight = "font-weight: bold;"
		default:
			fontSize = "font-size: small;"
			fontWeight = "font-weight: bold;"
		}
		
		return fmt.Sprintf(`<div %s %s %s>%s</div>`, gmailParagraphStyle, fontSize, fontWeight, content)
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
