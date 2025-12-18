package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Kullanım: gowebscraper <url>")
		os.Exit(1)
	}

	targetURL := os.Args[1]
	fmt.Printf("Scraping başlatılıyor: %s\n", targetURL)

	dirName, err := createDirName(targetURL)
	if err != nil {
		fmt.Printf("Dizin adı oluşturulurken hata: %v\n", err)
		os.Exit(1)
	}

	// izin vermeyince sorun yarattı
	err = os.MkdirAll(dirName, 0755)
	if err != nil {
		fmt.Printf("Dizin oluşturulurken hata: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Dizin oluşturuldu: %s\n", dirName)

	htmlContent, err := fetchHTML(targetURL)
	if err != nil {
		fmt.Printf("HTML çekilirken hata: %v\n", err)
		os.Exit(1)
	}

	htmlPath := filepath.Join(dirName, "site_data.html")
	err = saveHTML(htmlContent, htmlPath)
	if err != nil {
		fmt.Printf("HTML kaydedilirken hata: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("HTML içeriği %s dosyasına kaydedildi\n", htmlPath)

	urls := extractURLs(htmlContent, targetURL)
	urlsPath := filepath.Join(dirName, "urls.txt")
	err = saveURLs(urls, urlsPath)
	if err != nil {
		fmt.Printf("URL'ler kaydedilirken hata: %v\n", err)
	} else {
		fmt.Printf("%d URL bulundu, %s dosyasına kaydedildi\n", len(urls), urlsPath)
	}

	screenshotPath := filepath.Join(dirName, "screenshot.png")
	err = takeScreenshot(targetURL, screenshotPath)
	if err != nil {
		fmt.Printf("Ekran görüntüsü alınırken hata: %v\n", err)
	} else {
		fmt.Printf("Ekran görüntüsü %s dosyasına kaydedildi\n", screenshotPath)
	}

	fmt.Println("Scraping başarıyla tamamlandı!")
}

func fetchHTML(url string) (string, error) {
	// timeout 30 saniye yeterli olur diye düşündüm
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// siteler bot isteklerini engelliyor diye ekledim
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// 200 dışında bir kod gelirse hata veriyoruz
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

func saveHTML(content, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

func extractURLs(html, baseURL string) []string {
	var urls []string
	// aynı url'yi iki kere eklememek için map kullandım
	urlMap := make(map[string]bool)

	// regex ile href içindeki linkleri buluyorum, farklı yolu var mı bilmiyorum daha sonra dönüş yapacağım
	hrefPattern := regexp.MustCompile(`href=["']([^"']+)["']`)
	matches := hrefPattern.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) > 1 {
			url := match[1]

			if strings.HasPrefix(url, "/") {
				url = baseURL + url
			} else if !strings.HasPrefix(url, "http") {
				url = baseURL + "/" + url
			}

			if !urlMap[url] {
				urls = append(urls, url)
				urlMap[url] = true
			}
		}
	}

	return urls
}

func saveURLs(urls []string, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, url := range urls {
		_, err = file.WriteString(url + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

// chrome ile screenshot almak için, choromium tabanlı tüm tarayıclarda çalışıyormu bilmiyorum başka tarayıcılar için farklı kodlar yazabilirim daha sonra dönüş yapacağım
func takeScreenshot(url, filename string) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var buf []byte
	var height float64

	// sayfanın tamamını ekrangörüntüsü alamıyordum ve mobil görünümde gösteriyordu böyle yapınca düzeldi
	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1920, 1080), // deger vermeden düzenlebilirmi bakıcam daha sonra
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`Math.max(
			document.body.scrollHeight,
			document.body.offsetHeight,
			document.documentElement.clientHeight,
			document.documentElement.scrollHeight,
			document.documentElement.offsetHeight
		)`, &height),
	)
	if err != nil {
		return err
	}

	// viewportu desktop genişliğinde ama sayfanın tam yüksekliğinde ayarlıyoruz, burayada daha sonra bakacağım
	err = chromedp.Run(ctx,
		chromedp.EmulateViewport(1920, int64(height)),
		chromedp.Sleep(1*time.Second),
		chromedp.CaptureScreenshot(&buf),
	)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, buf, 0644)
}

func createDirName(targetURL string) (string, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return "", err
	}

	host := parsedURL.Host
	host = strings.TrimPrefix(host, "www.")

	// özel karakterler sorun yarattı
	dirName := strings.ReplaceAll(host, ".", "_")
	dirName = strings.ReplaceAll(dirName, "-", "_")

	if parsedURL.Path != "" && parsedURL.Path != "/" {
		pathPart := strings.Trim(parsedURL.Path, "/")
		pathPart = strings.ReplaceAll(pathPart, "/", "_")
		pathPart = strings.ReplaceAll(pathPart, "-", "_")
		if pathPart != "" {
			dirName = dirName + "_" + pathPart
		}
	}

	return dirName, nil
}
