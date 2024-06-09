module.exports = {
  content: ["./internal/**/*.templ", "./internal/**/*.html"],

  plugins: [require("daisyui")],

  daisyui: {
    themes: ["default", "retro", "cyberpunk", "valentine", "aqua", "nord"],
  },
};
