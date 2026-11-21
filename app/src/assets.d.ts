// Metro resolves an imported image to an asset ID for <Image source>.
declare module '*.png' {
  const source: number;
  export default source;
}
