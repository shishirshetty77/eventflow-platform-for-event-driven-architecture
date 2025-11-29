/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  env: {
    API_URL: process.env.API_URL || 'http://localhost:8007',
    WS_URL: process.env.WS_URL || 'ws://localhost:8007/ws',
  },
}

module.exports = nextConfig
