const setStyle = (target: any, style: any): void => {
  if (!target || !target.style) {
    return;
  }
  Object.keys(style).forEach((key) => {
    target.style[key] = style[key];
  });
};

export default {
  setStyle,
};
