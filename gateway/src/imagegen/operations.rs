use super::{adapter_kind, provider_error, AdapterConfig, AdapterKind};
use anyhow::{anyhow, Context};
use image::{DynamicImage, GenericImageView, ImageBuffer, ImageFormat, Luma, Rgba};
use serde::{Deserialize, Serialize};
use std::io::Cursor;

pub(super) const MAX_INPUT_IMAGE_BYTES: usize = 32 * 1024 * 1024;
pub(super) const MAX_MASK_BYTES: usize = 8 * 1024 * 1024;
pub(super) const MAX_IMAGE_PIXELS: u64 = 25_000_000;

#[derive(Debug, Clone, Copy, Deserialize, Serialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub(crate) enum ImageOperation {
    Generate,
    Edit,
    Inpaint,
    ImageTransform,
    Variation,
    ReferenceGenerate,
    Outpaint,
}

impl ImageOperation {
    pub(crate) fn as_str(self) -> &'static str {
        match self {
            Self::Generate => "generate",
            Self::Edit => "edit",
            Self::Inpaint => "inpaint",
            Self::ImageTransform => "image_transform",
            Self::Variation => "variation",
            Self::ReferenceGenerate => "reference_generate",
            Self::Outpaint => "outpaint",
        }
    }
}

#[derive(Debug, Clone)]
pub(crate) struct CanonicalImage {
    pub png: Vec<u8>,
    pub width: u32,
    pub height: u32,
}

#[derive(Debug, Clone)]
pub(crate) struct MaskRaster {
    pub png: Vec<u8>,
    pub width: u32,
    pub height: u32,
    pub sha256: String,
    pub soft_edges: bool,
}

#[derive(Debug, Clone)]
pub(crate) struct NativeImageRequest {
    pub operation: ImageOperation,
    pub provider: String,
    pub model: String,
    pub endpoint_id: String,
    pub prompt: String,
    pub negative_prompt: String,
    pub source: Option<CanonicalImage>,
    pub mask: Option<MaskRaster>,
    pub size: String,
    pub quality: String,
    pub output_format: String,
    pub idempotency_key: String,
}

pub(crate) fn canonicalize_source(bytes: &[u8]) -> anyhow::Result<CanonicalImage> {
    if bytes.is_empty() || bytes.len() > MAX_INPUT_IMAGE_BYTES {
        return Err(anyhow!("source image must be between 1 byte and 32 MiB"));
    }
    let image = image::load_from_memory(bytes).context("decoding source image")?;
    let (width, height) = image.dimensions();
    validate_dimensions(width, height, "source image")?;
    let png = encode_rgba_png(&image.to_rgba8())?;
    Ok(CanonicalImage { png, width, height })
}

pub(crate) fn normalize_mask(bytes: &[u8]) -> anyhow::Result<MaskRaster> {
    if bytes.is_empty() || bytes.len() > MAX_MASK_BYTES {
        return Err(anyhow!("mask must be between 1 byte and 8 MiB"));
    }
    if !bytes.starts_with(b"\x89PNG\r\n\x1a\n") {
        return Err(anyhow!("mask must be a PNG image"));
    }
    let image = image::load_from_memory_with_format(bytes, ImageFormat::Png)
        .context("decoding mask PNG")?;
    let (width, height) = image.dimensions();
    validate_dimensions(width, height, "mask")?;
    let luma = image.to_luma8();
    let soft_edges = luma.iter().any(|value| !matches!(*value, 0 | 255));
    let png = encode_luma_png(&luma)?;
    let sha256 = sha256_hex(&png);
    Ok(MaskRaster {
        png,
        width,
        height,
        sha256,
        soft_edges,
    })
}

pub(super) fn alpha_edit_mask(mask: &MaskRaster) -> anyhow::Result<Vec<u8>> {
    let luma = image::load_from_memory_with_format(&mask.png, ImageFormat::Png)
        .context("decoding normalized mask")?
        .to_luma8();
    let rgba = ImageBuffer::<Rgba<u8>, Vec<u8>>::from_fn(mask.width, mask.height, |x, y| {
        let coverage = luma.get_pixel(x, y).0[0];
        Rgba([255, 255, 255, 255_u8.saturating_sub(coverage)])
    });
    encode_rgba_png(&rgba)
}

pub(crate) fn validate_native_request(
    config: &AdapterConfig,
    request: &NativeImageRequest,
) -> anyhow::Result<()> {
    if request.prompt.trim().is_empty() {
        return Err(provider_error(
            &request.provider,
            "INVALID_REQUEST",
            "prompt is required",
            false,
        ));
    }
    match request.operation {
        ImageOperation::Edit => {
            require_source(request)?;
            if request.mask.is_some() {
                return Err(provider_error(
                    &request.provider,
                    "INVALID_MASK",
                    "edit forbids a raster mask; use inpaint",
                    false,
                ));
            }
        }
        ImageOperation::Inpaint => {
            let source = require_source(request)?;
            let mask = request.mask.as_ref().ok_or_else(|| {
                provider_error(
                    &request.provider,
                    "INVALID_MASK",
                    "inpaint requires an edit-coverage mask",
                    false,
                )
            })?;
            if source.width != mask.width || source.height != mask.height {
                return Err(provider_error(
                    &request.provider,
                    "MASK_DIMENSION_MISMATCH",
                    format!(
                        "source is {}x{} but mask is {}x{}",
                        source.width, source.height, mask.width, mask.height
                    ),
                    false,
                ));
            }
        }
        _ => {
            return Err(capability_unsupported(
                request,
                "operation is not implemented",
            ));
        }
    }

    if !request.negative_prompt.trim().is_empty()
        && matches!(
            request.provider.as_str(),
            "openai" | "azure-openai" | "codex-oauth" | "gemini"
        )
    {
        return Err(provider_error(
            &request.provider,
            "CAPABILITY_UNSUPPORTED",
            "negative_prompt is not supported by this edit endpoint",
            false,
        ));
    }
    let kind = adapter_kind(&request.provider).ok_or_else(|| {
        provider_error(
            &request.provider,
            "CAPABILITY_UNSUPPORTED",
            "unknown image provider",
            false,
        )
    })?;
    let direct = super::provider_config(config, &request.provider);
    let missing = match kind {
        AdapterKind::CodexOAuth => {
            super::validate_bridge_endpoint(&config.bridge_url, &config.bridge_token).err()
        }
        _ if direct.base_url.trim().is_empty() => Some("base URL".to_string()),
        _ if direct.api_key.trim().is_empty() => Some("server-side API key".to_string()),
        _ => None,
    };
    if let Some(missing) = missing {
        return Err(provider_error(
            &request.provider,
            "AUTH_MISSING",
            format!("provider is not configured: missing {missing}"),
            false,
        ));
    }
    let endpoint_ok = request.endpoint_id.trim().is_empty()
        || match kind {
            AdapterKind::CodexOAuth => request.endpoint_id == "/v1/images/edits",
            AdapterKind::OpenAi => request.endpoint_id == "/images/edits",
            AdapterKind::AzureOpenAi => request.endpoint_id == "/images/edits",
            AdapterKind::Gemini => request.endpoint_id == "v1beta/interactions",
            AdapterKind::Stability => request.endpoint_id == "/stable-image/edit/inpaint",
            _ => false,
        };
    if !endpoint_ok {
        return Err(capability_unsupported(
            request,
            "endpoint is not allowlisted for this operation",
        ));
    }
    match (kind, request.operation) {
        (AdapterKind::CodexOAuth, ImageOperation::Edit) => Ok(()),
        (AdapterKind::CodexOAuth, ImageOperation::Inpaint) => Err(capability_unsupported(
            request,
            "Codex OAuth providers do not advertise raster masks",
        )),
        // The configured provider and its explicit edit endpoint establish this
        // capability. Do not guess it from a model name.
        (AdapterKind::Gemini, ImageOperation::Edit) => Ok(()),
        (AdapterKind::Gemini, ImageOperation::Inpaint) => Err(capability_unsupported(
            request,
            "Gemini semantic editing does not accept a raster mask",
        )),
        (AdapterKind::Stability, ImageOperation::Inpaint)
            if request.model == "stable-image-core" =>
        {
            Ok(())
        }
        (
            AdapterKind::OpenAi | AdapterKind::AzureOpenAi,
            ImageOperation::Edit | ImageOperation::Inpaint,
        ) if matches!(
            request
                .model
                .trim()
                .strip_prefix("openai/")
                .unwrap_or(request.model.trim()),
            "gpt-image-2" | "gpt-image-1.5" | "gpt-image-1" | "gpt-image-1-mini"
        ) =>
        {
            Ok(())
        }
        _ => Err(capability_unsupported(
            request,
            "provider/model/endpoint does not implement this operation",
        )),
    }
}

fn require_source(request: &NativeImageRequest) -> anyhow::Result<&CanonicalImage> {
    request.source.as_ref().ok_or_else(|| {
        provider_error(
            &request.provider,
            "INVALID_SOURCE_IMAGE",
            "operation requires an existing source version",
            false,
        )
    })
}

fn capability_unsupported(request: &NativeImageRequest, detail: &str) -> anyhow::Error {
    provider_error(
        &request.provider,
        "CAPABILITY_UNSUPPORTED",
        format!(
            "{}:{} does not support {}: {detail}",
            request.provider,
            request.model,
            request.operation.as_str()
        ),
        false,
    )
}

fn validate_dimensions(width: u32, height: u32, label: &str) -> anyhow::Result<()> {
    if width == 0 || height == 0 || u64::from(width) * u64::from(height) > MAX_IMAGE_PIXELS {
        return Err(anyhow!("{label} dimensions exceed the 25 megapixel limit"));
    }
    Ok(())
}

fn encode_rgba_png(image: &ImageBuffer<Rgba<u8>, Vec<u8>>) -> anyhow::Result<Vec<u8>> {
    let mut bytes = Cursor::new(Vec::new());
    DynamicImage::ImageRgba8(image.clone())
        .write_to(&mut bytes, ImageFormat::Png)
        .context("encoding canonical PNG")?;
    Ok(bytes.into_inner())
}

fn encode_luma_png(image: &ImageBuffer<Luma<u8>, Vec<u8>>) -> anyhow::Result<Vec<u8>> {
    let mut bytes = Cursor::new(Vec::new());
    DynamicImage::ImageLuma8(image.clone())
        .write_to(&mut bytes, ImageFormat::Png)
        .context("encoding normalized mask PNG")?;
    Ok(bytes.into_inner())
}

fn sha256_hex(bytes: &[u8]) -> String {
    use sha2::{Digest, Sha256};
    Sha256::digest(bytes)
        .iter()
        .map(|byte| format!("{byte:02x}"))
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn png_luma(values: &[u8], width: u32, height: u32) -> Vec<u8> {
        encode_luma_png(&ImageBuffer::from_raw(width, height, values.to_vec()).unwrap()).unwrap()
    }

    #[test]
    fn normalizes_coverage_and_converts_to_transparent_edit_alpha() {
        let mask = normalize_mask(&png_luma(&[0, 128, 255], 3, 1)).unwrap();
        assert!(mask.soft_edges);
        let alpha = image::load_from_memory(&alpha_edit_mask(&mask).unwrap())
            .unwrap()
            .to_rgba8();
        assert_eq!(alpha.get_pixel(0, 0).0[3], 255);
        assert_eq!(alpha.get_pixel(1, 0).0[3], 127);
        assert_eq!(alpha.get_pixel(2, 0).0[3], 0);
    }

    #[test]
    fn rejects_non_png_and_oversized_pixel_grids() {
        assert!(normalize_mask(b"not png").is_err());
        assert!(validate_dimensions(5_001, 5_001, "mask").is_err());
    }

    #[test]
    fn distinguishes_semantic_edits_from_raster_inpainting() {
        let source = CanonicalImage {
            png: png_luma(&[0], 1, 1),
            width: 1,
            height: 1,
        };
        let mask = normalize_mask(&png_luma(&[255], 1, 1)).unwrap();
        let mut providers = std::collections::HashMap::new();
        providers.insert(
            "gemini".to_string(),
            super::super::ProviderConfig {
                base_url: "https://example.test".to_string(),
                api_key: "configured".to_string(),
                auth_mode: "bearer".to_string(),
                capability_probe_url: String::new(),
                api_version: String::new(),
            },
        );
        let config = super::super::AdapterConfig {
            provider: "gemini".to_string(),
            map_icon_provider: String::new(),
            base_url: String::new(),
            api_key: String::new(),
            providers,
            openclaw_url: String::new(),
            bridge_url: String::new(),
            bridge_token: String::new(),
            bridge_provider: String::new(),
            bridge_map_icon_provider: String::new(),
            bridge_fallbacks: Vec::new(),
            bridge_fallback_policy: String::new(),
            bridge_compatibility: String::new(),
        };
        let mut request = NativeImageRequest {
            operation: ImageOperation::Edit,
            provider: "gemini".to_string(),
            model: "gemini-3.1-flash-image".to_string(),
            endpoint_id: "v1beta/interactions".to_string(),
            prompt: "replace the sky".to_string(),
            negative_prompt: String::new(),
            source: Some(source),
            mask: None,
            size: String::new(),
            quality: String::new(),
            output_format: "png".to_string(),
            idempotency_key: "test".to_string(),
        };
        validate_native_request(&config, &request).unwrap();
        request.operation = ImageOperation::Inpaint;
        request.mask = Some(mask);
        assert!(validate_native_request(&config, &request)
            .unwrap_err()
            .to_string()
            .contains("does not accept a raster mask"));
    }
}
