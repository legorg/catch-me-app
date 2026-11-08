import type { Bounds, Coordinates, Playground } from './api/generated/types.gen';

const EARTH_RADIUS_METERS = 6371008.8;
export const DEFAULT_CELL_SIZE_METERS = 2000;

export function isValidCellSize(size: unknown): size is number {
  return typeof size === 'number' && Number.isInteger(size) && size >= 100 && size <= 5000 && size % 100 === 0;
}

export function parseCellSizeKm(text: string): number | null {
  const value = text.trim().replace(',', '.');
  if (!/^\d+(?:\.\d+)?$/.test(value)) return null;
  const meters = Number(value) * 1000;
  const rounded = Math.round(meters);
  return Math.abs(meters - rounded) < 1e-6 && isValidCellSize(rounded) ? rounded : null;
}

// Preview only. The backend computes and stores the authoritative geometry.
export function previewPlayground(center: Coordinates, cellSizeMeters = DEFAULT_CELL_SIZE_METERS): Playground {
  if (!isValidCellSize(cellSizeMeters)) {
    throw new Error('Cell size must be 0.1–5 km in steps of 0.1 km.');
  }
  if (!Number.isFinite(center.latitude) || Math.abs(center.latitude) > 80 ||
      !Number.isFinite(center.longitude) || Math.abs(center.longitude) > 180) {
    throw new Error('Choose a center between 80° south and 80° north.');
  }
  const latitudeDelta = (6 * cellSizeMeters / 2) / EARTH_RADIUS_METERS * 180 / Math.PI;
  const longitudeDelta = latitudeDelta / Math.cos(center.latitude * Math.PI / 180);
  const bounds = {
    north: center.latitude + latitudeDelta, south: center.latitude - latitudeDelta,
    east: center.longitude + longitudeDelta, west: center.longitude - longitudeDelta,
  };
  if (bounds.west < -180 || bounds.east > 180) {
    throw new Error('The playground cannot cross the international date line.');
  }
  return { center, bounds, rows: 6, columns: 6, cellSizeMeters };
}

export function boundaryCoordinates(b: Bounds): Coordinates[] {
  return [
    { latitude: b.north, longitude: b.west },
    { latitude: b.north, longitude: b.east },
    { latitude: b.south, longitude: b.east },
    { latitude: b.south, longitude: b.west },
  ];
}

export function gridLines(p: Playground): Coordinates[][] {
  const { bounds: b, rows, columns } = p;
  const lines: Coordinates[][] = [];
  for (let row = 1; row < rows; row++) {
    const latitude = b.north - (b.north - b.south) * row / rows;
    lines.push([{ latitude, longitude: b.west }, { latitude, longitude: b.east }]);
  }
  for (let column = 1; column < columns; column++) {
    const longitude = b.west + (b.east - b.west) * column / columns;
    lines.push([{ latitude: b.north, longitude }, { latitude: b.south, longitude }]);
  }
  return lines;
}

export function cellLabels(p: Playground) {
  return Array.from({ length: p.rows * p.columns }, (_, index) => {
    const row = Math.floor(index / p.columns);
    const column = index % p.columns;
    return {
      label: `${String.fromCharCode(65 + column)}${row + 1}`,
      coordinate: {
        latitude: p.bounds.north - (p.bounds.north - p.bounds.south) * (row + 0.5) / p.rows,
        longitude: p.bounds.west + (p.bounds.east - p.bounds.west) * (column + 0.5) / p.columns,
      },
    };
  });
}

/** Bounds of a cell such as "C4" (columns A–F west to east, rows 1–6 north to south). */
export function cellBounds(p: Playground, label: string): Bounds {
  const column = label.charCodeAt(0) - 65;
  const row = Number(label.slice(1)) - 1;
  if (!/^[A-F][1-6]$/.test(label) || column >= p.columns || row >= p.rows) {
    throw new Error(`Unknown cell ${label}.`);
  }
  const height = (p.bounds.north - p.bounds.south) / p.rows;
  const width = (p.bounds.east - p.bounds.west) / p.columns;
  const north = p.bounds.north - height * row;
  const west = p.bounds.west + width * column;
  return { north, south: north - height, west, east: west + width };
}

/** Which playground edge a cell in row 1 or 6 lies on. */
export function edgeName(label: string): string {
  return label.endsWith('1') ? 'north edge' : label.endsWith('6') ? 'south edge' : 'inside';
}
