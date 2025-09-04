/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        'hack-red': '#ec3750',
        'hack-orange': '#ff8c37',
        'hack-yellow': '#f1c40f',
        'hack-green': '#33d9b2',
        'hack-cyan': '#3742fa',
        'hack-blue': '#70a1ff',
        'hack-purple': '#5352ed',
      },
      fontFamily: {
        'mono': ['SF Mono', 'Monaco', 'Consolas', 'Liberation Mono', 'Courier New', 'monospace'],
      },
    },
  },
  plugins: [],
}
