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

/** A whole number typed by the user, or null if it is not one or not valid. */
export function parseWholeNumber(text: string, isValid: (value: number) => boolean): number | null {
  const value = text.trim();
  return /^\d+$/.test(value) && isValid(Number(value)) ? Number(value) : null;
}

// Preview only. The backend computes and stores the authoritative geometry.
export function previewPlayground(center: Coordinates, cellSizeMeters = DEFAULT_CELL_SIZE_METERS, gridSize = DEFAULT_GRID_SIZE): Playground {
  if (!isValidCellSize(cellSizeMeters)) {
    throw new Error(translate('playground.cellSizeInvalid'));
  }
  if (!isValidGridSize(gridSize)) {
    throw new Error(translate('playground.gridSizeInvalid'));
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
  if (bounds.west < -180 || bounds.east > 180) {
    throw new Error(translate('playground.dateLine'));
  }
  return { center, bounds, rows: gridSize, columns: gridSize, cellSizeMeters };
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

/** Bounds of a cell such as "C4" (columns A, B, … west to east, rows 1, 2, … north to south). */
export function cellBounds(p: Playground, label: string): Bounds {
  const column = label.charCodeAt(0) - 65;
  const row = Number(label.slice(1)) - 1;
  if (!/^[A-L](?:[1-9]|1[0-2])$/.test(label) || column >= p.columns || row >= p.rows) {
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
  const row = Number(label.slice(1));
  return translate(row === 1 ? 'playground.northEdge' : row === p.rows ? 'playground.southEdge' : 'playground.inside');
}

/** Label of the cell containing the coordinates, or null outside the playground. Mirrors the backend. */
export function cellAt(p: Playground, c: Coordinates): string | null {
  const b = p.bounds;
  if (!(c.latitude <= b.north && c.latitude >= b.south && c.longitude >= b.west && c.longitude <= b.east)) {
    return null;
  }
  const row = Math.min(Math.floor((b.north - c.latitude) / (b.north - b.south) * p.rows), p.rows - 1);
  const column = Math.min(Math.floor((c.longitude - b.west) / (b.east - b.west) * p.columns), p.columns - 1);
  return `${String.fromCharCode(65 + column)}${row + 1}`;
}

/** Labels of the cells in one row of the grid (1 is the northern row). */
export function rowCells(p: Playground, row: number): string[] {
  return Array.from({ length: p.columns }, (_, column) => `${String.fromCharCode(65 + column)}${row}`);
}

/**
 * Which point of a chosen route the coordinates may be: the start in the southern edge row, the stop in the
 * northern one; null elsewhere or outside. Mirrors the backend.
 */
export function routePointAt(p: Playground, c: Coordinates): 'start' | 'stop' | null {
  const cell = cellAt(p, c);
  if (!cell) return null;
  const row = Number(cell.slice(1));
  return row === p.rows ? 'start' : row === 1 ? 'stop' : null;
}
