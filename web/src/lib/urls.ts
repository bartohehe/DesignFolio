/** Warianty pliku serwowane przez API (endpoint dochodzi w etapie 4). */
export type AssetVariant = 'original' | 'thumb' | 'preview'

export function assetUrl(assetId: string, variant: AssetVariant): string {
  return `/api/files/${encodeURIComponent(assetId)}/${variant}`
}
