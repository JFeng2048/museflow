/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{vue,ts,js,tsx,jsx}'],
  theme: {
    extend: {
      colors: {
        // 语义色板：映射到 CSS 变量，随主题切换
        ink: {
          DEFAULT: 'var(--c-ink)',
          soft: 'var(--c-ink-soft)',
          muted: 'var(--c-ink-muted)',
        },
        amber: {
          DEFAULT: 'var(--c-amber)',
          deep: 'var(--c-amber-deep)',
          soft: 'var(--c-amber-soft)',
        },
        warm: {
          DEFAULT: 'var(--c-warm)',
          '2': 'var(--c-warm-2)',
        },
        paper: 'var(--c-paper)',
        line: {
          DEFAULT: 'var(--c-line)',
          soft: 'var(--c-line-soft)',
        },
        success: 'var(--c-success)',
        warn: 'var(--c-warn)',
        danger: 'var(--c-danger)',
        info: 'var(--c-info)',
      },
      fontFamily: {
        serif: 'var(--font-serif)',
        sans: 'var(--font-sans)',
      },
      borderRadius: {
        s: 'var(--radius-s)',
        m: 'var(--radius-m)',
        l: 'var(--radius-l)',
      },
      boxShadow: {
        sm: 'var(--shadow-sm)',
        m: 'var(--shadow-m)',
        l: 'var(--shadow-l)',
      },
      spacing: {
        gap: 'var(--gap)',
        'gap-l': 'var(--gap-l)',
      },
    },
  },
  plugins: [],
}
