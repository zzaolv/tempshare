
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'brand-bg': '#F5F5F5',
        'brand-cyan': '#2BD8FB',
        'brand-yellow': '#EEF864',
        'brand-mint': '#64F8C5',
        'brand-dark': '#2c3e50',
        'brand-light': '#34495e',
      },
      boxShadow: {
        'soft-lg': '0 10px 25px -3px rgba(0, 0, 0, 0.05), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
        'soft-xl': '0 20px 40px -5px rgba(0, 0, 0, 0.08)',
        'soft-2xl': '0 25px 50px -12px rgba(0, 0, 0, 0.1)',
      },
      backdropBlur: {
        'xl': '24px',
        '2xl': '40px',
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
