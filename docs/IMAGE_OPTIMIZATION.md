# Image Optimization

Stormkit includes optional image optimization capabilities that can resize and optimize images on-the-fly. This feature is **opt-in** and requires additional system dependencies.

## Enabling Image Optimization

Image optimization is controlled at **build time** using Go build tags. By default, image optimization is **disabled** to simplify deployment and avoid requiring additional system dependencies.

### Building with Image Optimization Enabled

To enable image optimization, run the hosting service with the `imageopt` build tag:

```yml
# ProcFile
hosting: go run -tags=imageopt src/ee/hosting/main.go
```

## System Dependencies

When image optimization is enabled (with the `imageopt` build tag), the following system dependencies are required:

### macOS

```bash
brew install vips pkg-config
```

### Ubuntu/Debian

```bash
apt-get update
apt-get install -y libvips-dev pkg-config
```

### Alpine Linux

```bash
apk add --no-cache vips-dev pkgconf
```

## How It Works

The image optimization feature uses Go build tags to conditionally compile different implementations:

- **With `imageopt` tag**: Uses [bimg](https://github.com/h2non/bimg) library (which wraps libvips) for high-performance image processing
- **Without `imageopt` tag**: Uses a no-op implementation that returns an error when image optimization is requested

When image optimization is disabled:

- Images with `?size=` query parameters will be served without optimization
- No errors are thrown - the original image is served instead
- The build does not require libvips or any image processing dependencies

## Usage

Once built with image optimization enabled, users can request optimized images by adding query parameters:

```
# Resize to 300px width, auto height
GET /images/photo.jpg?size=300

# Resize to 300x200
GET /images/photo.jpg?size=300x200

# Smart crop to 300x200
GET /images/photo.jpg?size=300x200&smart=true
```

## Performance

- Optimized images are cached in Redis for 24 hours
- Maximum of 5 variants per image to prevent abuse
- Maximum image size is limited to 2048 pixels (width or height)
