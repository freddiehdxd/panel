/** @type {import('next').NextConfig} */
const nextConfig = {
  // All pages fetch live data from the backend — disable static caching
  fetchCache: 'force-no-store',

  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${process.env.BACKEND_URL ?? 'http://127.0.0.1:4000'}/api/:path*`,
      },
    ];
  },

  async headers() {
    return [
      {
        // Prevent browser caching of API responses proxied through Next.js
        source: '/api/:path*',
        headers: [
          { key: 'Cache-Control', value: 'no-store' },
        ],
      },
    ];
  },
};

module.exports = nextConfig;
