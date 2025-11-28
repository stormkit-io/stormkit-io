---
title: Framework Agnostic Image Optimization
description: Improve your website performance with Stormkit's framework-agnostic image optimization. Resize, crop, and intelligently focus images using simple URL parameters. Easy implementation with built-in caching and security.
---

# Image Optimization for Stormkit

Stormkit includes a powerful, framework-agnostic image optimization feature that allows you to resize and crop images on-the-fly through simple URL parameters. This feature helps improve your website's performance by delivering appropriately sized images to your users.

## Overview

The image optimization service automatically processes images when specific query parameters are detected in the image URL. Optimized images are cached for improved performance on subsequent requests.

## Usage

To optimize an image, simply add query parameters to your image URL:

```
https://your-domain.com/path/to/image.jpg?size=WIDTHxHEIGHT&smart=true|false
```

### Parameters

| Parameter | Description                                                                  | Format                         | Example                      |
| --------- | ---------------------------------------------------------------------------- | ------------------------------ | ---------------------------- |
| `size`    | Specifies the desired dimensions                                             | `WIDTHxHEIGHT` or just `WIDTH` | `size=400x300` or `size=400` |
| `smart`   | Enables smart cropping to maintain focus on the important parts of the image | `true` or `false`              | `smart=true`                 |

### Examples

1. **Basic Resize (Maintaining Aspect Ratio)**

   ```
   https://your-domain.com/images/photo.jpg?size=400
   ```

   This will resize the image to 400px width while maintaining its aspect ratio.

2. **Specific Dimensions**

   ```
   https://your-domain.com/images/photo.jpg?size=400x300
   ```

   This will force the image to be exactly 400px wide and 300px tall.

3. **Smart Cropping**
   ```
   https://your-domain.com/images/photo.jpg?size=400x300&smart=true
   ```
   This will crop the image to 400x300px dimensions while intelligently focusing on the important parts of the image.

## Security and Limitations

- **Maximum Dimensions**: For security reasons, the maximum allowed dimension is 2048px (width or height). Any request exceeding this limit will return the original image.
- **Variant Limit**: Each original image can have a maximum of 5 optimized variants to prevent storage abuse. Additional variant requests beyond this limit will return the original image.
- **Caching**: Optimized images are cached for 24 hours for improved performance.

## Best Practices

1. **Responsive Images**: Use different size parameters based on the user's device to deliver appropriate images.

   ```html
   <picture>
     <source media="(max-width: 600px)" srcset="/images/photo.jpg?size=600" />
     <source media="(max-width: 1200px)" srcset="/images/photo.jpg?size=1200" />
     <img src="/images/photo.jpg?size=2048" alt="Description" />
   </picture>
   ```

2. **Smart Cropping for Thumbnails**: When creating thumbnails or profile pictures, use smart cropping to ensure the subject remains in focus.

   ```
   /images/profile.jpg?size=150x150&smart=true
   ```

## Error Handling

If an image cannot be optimized due to any reason (invalid image format, processing error, etc.), the system will return the original image without optimization.

## Supported Formats

The image optimization service supports the following formats:

- JPEG
- PNG
- WebP
- GIF (non-animated)
- TIFF

## Need Help?

If you encounter any issues with the image optimization feature, please contact Stormkit support or open an issue on our GitHub repository.
