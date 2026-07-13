interface Props {
  apiFiles?: APIFile[];
  cdnFiles?: CDNFile[];
  functionHandler?: string;
}

export default ({
  apiFiles,
  cdnFiles,
  functionHandler = "index.mjs:handler",
}: Props = {}): Manifest => ({
  apiFiles,
  cdnFiles: cdnFiles || [
    {
      fileName: "/manifest.json",
      headers: { ETag: '"e30-ix3X3IswzLsliBTkCjpm3pS6818"' },
    },
    {
      fileName: "/_nuxt/default.6339aee9.mjs",
      headers: { ETag: '"3bc-1IuADqQAFBHoF9kID5HyUDAn5nM"' },
    },
    {
      fileName: "/_nuxt/entry.258d4c3a.css",
      headers: { ETag: '"18e5-cCou1R4OlQImO3wbWRFAa94/jZQ"' },
    },
    {
      fileName: "/_nuxt/entry.2fba3867.mjs",
      headers: { ETag: '"1d43f-ShdhnQKAsrgV5kMQqp5rF+uhLzs"' },
    },
    {
      fileName: "/_nuxt/error-404.042bd6ca.css",
      headers: { ETag: '"e34-OTUl57CUZBt2TaCSF5b2F82Bvjw"' },
    },
    {
      fileName: "/_nuxt/error-404.ce51d756.mjs",
      headers: { ETag: '"8a4-MF/b96Wxf1oBTnMMgMBSmbp9QV8"' },
    },
    {
      fileName: "/_nuxt/error-500.de29328c.css",
      headers: { ETag: '"7a4-bSbBvgcAYQhoLzjuAMnAHF9lE5s"' },
    },
    {
      fileName: "/_nuxt/error-500.de409a80.mjs",
      headers: { ETag: '"752-gXH50fZYva4+BnaxKDuNt50JdVo"' },
    },
    {
      fileName: "/_nuxt/error-component.3e032c47.mjs",
      headers: { ETag: '"44f-K7uwv/XuTY+sTbRN0cKC3oYvI5k"' },
    },
    {
      fileName: "/_nuxt/index.ae617684.mjs",
      headers: { ETag: '"70b-K0SPPoDglpNeJGuew0o5otYOpSE"' },
    },
  ],
  redirects: null,
  functionHandler: functionHandler,
});
