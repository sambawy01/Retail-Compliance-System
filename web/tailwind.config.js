/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        bg: {
          DEFAULT: '#0f1115',
          card: '#171a21',
          hover: '#1e222c',
        },
        border: {
          DEFAULT: '#2a2e3a',
          light: '#3a3f4d',
        },
        accent: {
          DEFAULT: '#3b82f6',
          hover: '#2563eb',
        },
        critical: '#ef4444',
        warning: '#f59e0b',
        info: '#3b82f6',
        success: '#22c55e',
        degraded: '#f59e0b',
        offline: '#6b7280',
        text: {
          primary: '#e5e7eb',
          secondary: '#9ca3af',
          muted: '#6b7280',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
}