# Go Web Scraper

Go dili ile yazılmış basit bir web scraper uygulaması.

## Özellikler

- Web sayfalarından HTML içeriği çekme
- HTML içeriğini dosyaya kaydetme
- Sayfadaki URL'leri bulma ve listeleme
- Ekran görüntüsü alma

## Kurulum

```bash
go mod download
go build -o gowebscraper
```

## Kullanım

```bash
./gowebscraper <url>
```

Örnek:

```bash
./gowebscraper https://example.com
```

## Çıktı Dosyaları

- `site_data.html` - Çekilen HTML içeriği
- `urls.txt` - Bulunan URL'lerin listesi
- `screenshot.png` - Sayfanın ekran görüntüsü

## Gereksinimler

- Go 1.25.5 veya üzeri
- Chrome/Chromium (screenshot için)
