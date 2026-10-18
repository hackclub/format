/** @type {import('next').NextConfig} */
const nextConfig = {
  turbopack: { root: __dirname },
  outputFileTracingRoot: __dirname,
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:8080/api/:path*',
      },
    ]
  },
  images: {
    remotePatterns: [{ protocol: 'https', hostname: 'i.format.hackclub.com' }],
  },
}

module.exports = nextConfig
