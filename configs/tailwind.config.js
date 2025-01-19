module.exports = {
  content: ["./internal/**/*.html"],

  plugins: [
    require("daisyui"),
    require("postcss-import")
  ],

  daisyui: {
    themes: [
      {
        dark: {
          ...require("daisyui/src/theming/themes")["dark"],
          fontFamily: "ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,Liberation Mono,Courier New,monospace",
        },
      }, 
    ],
  },
};
