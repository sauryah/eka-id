/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  ...(process.env.NEXT_EXPORT === 'true' ? { output: 'export' } : {}),
  images: {
    unoptimized: true,
  },
};

module.exports = nextConfig;