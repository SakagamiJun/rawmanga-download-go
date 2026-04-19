export default {
    darkMode: ["class", '[data-theme="dark"]'],
    content: ["./index.html", "./src/**/*.{ts,tsx}"],
    theme: {
        extend: {
            colors: {
                background: "rgb(var(--bg) / <alpha-value>)",
                foreground: "rgb(var(--fg) / <alpha-value>)",
                muted: "rgb(var(--muted) / <alpha-value>)",
                "muted-foreground": "rgb(var(--muted-fg) / <alpha-value>)",
                primary: "rgb(var(--primary) / <alpha-value>)",
                "primary-foreground": "rgb(var(--primary-fg) / <alpha-value>)",
                card: "rgb(var(--card) / <alpha-value>)",
                border: "rgb(var(--border) / <alpha-value>)",
                success: "rgb(var(--success) / <alpha-value>)",
                warning: "rgb(var(--warning) / <alpha-value>)",
                danger: "rgb(var(--danger) / <alpha-value>)"
            },
            boxShadow: {
                glow: "0 20px 70px rgba(20, 87, 180, 0.25)"
            }
        }
    },
    plugins: [],
};
