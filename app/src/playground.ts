import type { Bounds, Coordinates, Playground } from './api/types';
import { translate } from './i18n';

const EARTH_RADIUS_METERS = 6371008.8;
export const DEFAULT_CELL_SIZE_METERS = 2000;
/** Cells per side of the square grid, as the backend allows. */
export const DEFAULT_GRID_SIZE = 6;
export const MIN_GRID_SIZE = 3;
export const MAX_GRID_SIZE = 12;
/** Minutes between two reveals, as the backend allows. */
export const DEFAULT_REVEAL_INTERVAL_MINUTES = 30;
export const MIN_REVEAL_INTERVAL_MINUTES = 1;
export const MAX_REVEAL_INTERVAL_MINUTES = 60;
/** Where the draft playground starts before the user drags it (Cologne). */
export const DEFAULT_CENTER: Coordinates = { latitude: 50.9375, longitude: 6.9603 };
/** Shared start/finish landmark radius; the backend enforces the same distance. */
export const START_RADIUS_METERS = 50;

/** Great-circle (haversine) distance in meters. */
export function distanceMeters(a: Coordinates, b: Coordinates): number {
  const rad = Math.PI / 180;
  const dLat = (b.latitude - a.latitude) * rad;
  const dLon = (b.longitude - a.longitude) * rad;
  const h = Math.sin(dLat / 2) ** 2 + Math.cos(a.latitude * rad) * Math.cos(b.latitude * rad) * Math.sin(dLon / 2) ** 2;
  return 2 * EARTH_RADIUS_METERS * Math.asin(Math.sqrt(h));
}

/** The point `meters` away from `from` in the direction `bearingDegrees` (clockwise from north), on a local flat approximation. */
export function offsetPoint(from: Coordinates, bearingDegrees: number, meters: number): Coordinates {
  const rad = Math.PI / 180;
  const angular = meters / EARTH_RADIUS_METERS / rad;
  return {
    latitude: from.latitude + angular * Math.cos(bearingDegrees * rad),
    longitude: from.longitude + angular * Math.sin(bearingDegrees * rad) / Math.cos(from.latitude * rad),
  };
}

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

export function isValidGridSize(cells: unknown): cells is number {
  return Number.isInteger(cells) && (cells as number) >= MIN_GRID_SIZE && (cells as number) <= MAX_GRID_SIZE;
}

export function isValidRevealInterval(minutes: unknown): minutes is number {
  return Number.isInteger(minutes) && (minutes as number) >= MIN_REVEAL_INTERVAL_MINUTES && (minutes as number) <= MAX_REVEAL_INTERVAL_MINUTES;
}

/** Whole degrees the grid is turned clockwise, as the backend allows. */
export function isValidRotation(degrees: unknown): degrees is number {
  return Number.isInteger(degrees) && (degrees as number) >= 0 && (degrees as number) < 360;
}

/** The grid rotation that puts row A on top of a map whose top points to heading (degrees clockwise from north). */
export function rotationFromHeading(heading: number): number {
  return ((Math.round(heading) % 360) + 360) % 360;
}

/** A whole number typed by the user, or null if it is not one or not valid. */
export function parseWholeNumber(text: string, isValid: (value: number) => boolean): number | null {
  const value = text.trim();
  return /^\d+$/.test(value) && isValid(Number(value)) ? Number(value) : null;
}

// Preview only. The backend computes and stores the authoritative geometry.
export function previewPlayground(center: Coordinates, cellSizeMeters = DEFAULT_CELL_SIZE_METERS, gridSize = DEFAULT_GRID_SIZE, rotationDegrees = 0): Playground {
  if (!isValidCellSize(cellSizeMeters)) {
    throw new Error(translate('playground.cellSizeInvalid'));
  }
  if (!isValidGridSize(gridSize)) {
    throw new Error(translate('playground.gridSizeInvalid'));
  }
  if (!isValidRotation(rotationDegrees)) {
    throw new Error(translate('playground.rotationInvalid'));
  }
  if (!Number.isFinite(center.latitude) || Math.abs(center.latitude) > 80 ||
      !Number.isFinite(center.longitude) || Math.abs(center.longitude) > 180) {
    throw new Error(translate('playground.latitudeInvalid'));
  }
  const latitudeDelta = (gridSize * cellSizeMeters / 2) / EARTH_RADIUS_METERS * 180 / Math.PI;
  const longitudeDelta = latitudeDelta / Math.cos(center.latitude * Math.PI / 180);
  const bounds = {
    north: center.latitude + latitudeDelta, south: center.latitude - latitudeDelta,
    east: center.longitude + longitudeDelta, west: center.longitude - longitudeDelta,
  };
  const playground = { center, bounds, rows: gridSize, columns: gridSize, cellSizeMeters, rotationDegrees };
  const longitudes = playgroundCorners(playground).map((c) => c.longitude);
  if (Math.min(...longitudes) < -180 || Math.max(...longitudes) > 180) {
    throw new Error(translate('playground.dateLine'));
  }
  return playground;
}

/**
 * Turns a point of the unrotated grid clockwise around the center onto the map (or, with negative degrees, back).
 * Uses the same local flat approximation as the backend, so 0 degrees returns the point unchanged.
 */
function turn(p: Playground, c: Coordinates, clockwiseDegrees: number): Coordinates {
  if (clockwiseDegrees % 360 === 0) return c;
  const scale = Math.cos(p.center.latitude * Math.PI / 180);
  const east = (c.longitude - p.center.longitude) * scale;
  const north = c.latitude - p.center.latitude;
  const sin = Math.sin(clockwiseDegrees * Math.PI / 180);
  const cos = Math.cos(clockwiseDegrees * Math.PI / 180);
  return {
    latitude: p.center.latitude - east * sin + north * cos,
    longitude: p.center.longitude + (east * cos + north * sin) / scale,
  };
}

/** The point on the map of a point of the unrotated grid. */
export function rotatePoint(p: Playground, c: Coordinates): Coordinates {
  return turn(p, c, p.rotationDegrees);
}

/** The four corners of a north-aligned rectangle. */
export function boundaryCoordinates(b: Bounds): Coordinates[] {
  return [
    { latitude: b.north, longitude: b.west },
    { latitude: b.north, longitude: b.east },
    { latitude: b.south, longitude: b.east },
    { latitude: b.south, longitude: b.west },
  ];
}

/** The playground's corners on the map, turned with the grid. */
export function playgroundCorners(p: Playground): Coordinates[] {
  return boundaryCoordinates(p.bounds).map((c) => rotatePoint(p, c));
}

/** The corners of a cell such as "C4" on the map, turned with the grid. */
export function cellCorners(p: Playground, label: string): Coordinates[] {
  return boundaryCoordinates(cellBounds(p, label)).map((c) => rotatePoint(p, c));
}

/** The middle of a cell such as "C4" on the map. */
export function cellCenter(p: Playground, label: string): Coordinates {
  const b = cellBounds(p, label);
  return rotatePoint(p, { latitude: (b.north + b.south) / 2, longitude: (b.east + b.west) / 2 });
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
  return lines.map((line) => line.map((c) => rotatePoint(p, c)));
}

export function cellLabels(p: Playground) {
  return Array.from({ length: p.rows * p.columns }, (_, index) => {
    const label = `${String.fromCharCode(65 + Math.floor(index / p.columns))}${index % p.columns + 1}`;
    return { label, coordinate: cellCenter(p, label) };
  });
}

/**
 * Bounds of a cell such as "C4" in the unrotated grid (rows A, B, … north to south, columns 1, 2, … west to east);
 * use cellCorners or cellCenter for the map.
 */
export function cellBounds(p: Playground, label: string): Bounds {
  const row = label.charCodeAt(0) - 65;
  const column = Number(label.slice(1)) - 1;
  if (!/^[A-L](?:[1-9]|1[0-2])$/.test(label) || row >= p.rows || column >= p.columns) {
    throw new Error(translate('playground.unknownCell', { cell: label }));
  }
  const height = (p.bounds.north - p.bounds.south) / p.rows;
  const width = (p.bounds.east - p.bounds.west) / p.columns;
  const north = p.bounds.north - height * row;
  const west = p.bounds.west + width * column;
  return { north, south: north - height, west, east: west + width };
}

/** Which playground edge a cell in the first or last row lies on. */
export function edgeName(p: Playground, label: string): string {
  const row = label.charCodeAt(0) - 64;
  return translate(row === 1 ? 'playground.northEdge' : row === p.rows ? 'playground.southEdge' : 'playground.inside');
}

/** Label of the cell containing the coordinates, or null outside the playground. Mirrors the backend. */
export function cellAt(p: Playground, point: Coordinates): string | null {
  const b = p.bounds;
  // Turned back into the unrotated grid, where cells are north-aligned rectangles.
  const c = turn(p, point, -p.rotationDegrees);
  if (!(c.latitude <= b.north && c.latitude >= b.south && c.longitude >= b.west && c.longitude <= b.east)) {
    return null;
  }
  const row = Math.min(Math.floor((b.north - c.latitude) / (b.north - b.south) * p.rows), p.rows - 1);
  const column = Math.min(Math.floor((c.longitude - b.west) / (b.east - b.west) * p.columns), p.columns - 1);
  return `${String.fromCharCode(65 + row)}${column + 1}`;
}

/** Labels of the cells in one row of the grid (1 is the northern row). */
export function rowCells(p: Playground, row: number): string[] {
  return Array.from({ length: p.columns }, (_, column) => `${String.fromCharCode(64 + row)}${column + 1}`);
}

/**
 * Which point of a chosen route the coordinates may be: the start in the southern edge row, the stop in the
 * northern one; null elsewhere or outside. Mirrors the backend.
 */
export function routePointAt(p: Playground, c: Coordinates): 'start' | 'stop' | null {
  const cell = cellAt(p, c);
  if (!cell) return null;
  const row = cell.charCodeAt(0) - 64;
  return row === p.rows ? 'start' : row === 1 ? 'stop' : null;
}
