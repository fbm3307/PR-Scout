export function ghAvatar(username: string, size = 48): string {
  const name = username.endsWith('[bot]') ? username.slice(0, -5) : username;
  return `https://github.com/${name}.png?size=${size}`;
}
