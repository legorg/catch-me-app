import assert from 'node:assert/strict';
import { test } from 'node:test';

import {
  boundaryCoordinates, cellAt, cellBounds, cellCenter, cellCorners, cellLabels, distanceMeters, edgeName, gridLines, isValidGridSize,
  isValidRevealInterval, parseCellSizeKm, parseWholeNumber, playgroundCorners, previewPlayground, routePointAt, rowCells,
} from '../src/playground';

test('a chosen route starts in the southern and stops in the northern edge row', () => {
  const p = previewPlayground({ latitude: 52.52, longitude: 13.405 }, 1000, 4);
  assert.deepEqual(rowCells(p, 1), ['A1', 'B1', 'C1', 'D1']);
  assert.deepEqual(rowCells(p, 4), ['A4', 'B4', 'C4', 'D4']);
  const centers = new Map(cellLabels(p).map(({ label, coordinate }) => [label, coordinate]));
  assert.equal(routePointAt(p, centers.get('B4')!), 'start');
  assert.equal(routePointAt(p, centers.get('D1')!), 'stop');
  assert.equal(routePointAt(p, centers.get('C2')!), null);
  assert.equal(routePointAt(p, { latitude: p.bounds.south - 0.001, longitude: p.center.longitude }), null);
});

test('cellAt finds the labeled cell and rejects outside positions', () => {
  const p = previewPlayground({ latitude: 52.52, longitude: 13.405 }, 500);
  for (const { label, coordinate } of cellLabels(p)) {
    assert.equal(cellAt(p, coordinate), label);
  }
  assert.equal(cellAt(p, { latitude: p.bounds.south, longitude: p.bounds.east }), 'F6');
  assert.equal(cellAt(p, { latitude: p.bounds.north + 0.001, longitude: p.center.longitude }), null);
});

test('cell bounds match the labeled grid cells', () => {
  const p = previewPlayground({ latitude: 52.52, longitude: 13.405 });
  const labels = new Map(cellLabels(p).map(({ label, coordinate }) => [label, coordinate]));
  for (const label of ['A1', 'F1', 'C4', 'A6', 'F6']) {
    const b = cellBounds(p, label);
    const center = labels.get(label)!;
    assert.ok(Math.abs((b.north + b.south) / 2 - center.latitude) < 1e-12, label);
    assert.ok(Math.abs((b.east + b.west) / 2 - center.longitude) < 1e-12, label);
  }
  assert.equal(cellBounds(p, 'A1').north, p.bounds.north);
  assert.equal(cellBounds(p, 'A1').west, p.bounds.west);
  assert.ok(Math.abs(cellBounds(p, 'F6').south - p.bounds.south) < 1e-12);
  assert.ok(Math.abs(cellBounds(p, 'F6').east - p.bounds.east) < 1e-12);
  for (const label of ['G1', 'A7', 'A0', 'a1', '']) {
    assert.throws(() => cellBounds(p, label), /Unknown cell/);
  }
  assert.equal(edgeName(p, 'B1'), 'north edge');
  assert.equal(edgeName(p, 'E6'), 'south edge');
  assert.equal(edgeName(p, 'C3'), 'inside');
});

test('the grid size sets the rows, columns, labels and cells', () => {
  for (const cells of [3, 12]) {
    const p = previewPlayground({ latitude: 52.52, longitude: 13.405 }, 1000, cells);
    assert.equal(p.rows, cells);
    assert.equal(p.columns, cells);
    assert.ok(Math.abs((p.bounds.north - p.bounds.south) * Math.PI / 180 * 6371008.8 - cells * 1000) < 1e-6);
    const labels = cellLabels(p);
    assert.equal(labels.length, cells * cells);
    for (const { label, coordinate } of labels) {
      assert.equal(cellAt(p, coordinate), label);
      const b = cellBounds(p, label);
      assert.ok(coordinate.latitude < b.north && coordinate.latitude > b.south, label);
    }
  }
  const large = previewPlayground({ latitude: 52.52, longitude: 13.405 }, 1000, 12);
  assert.equal(cellAt(large, { latitude: large.bounds.south, longitude: large.bounds.east }), 'L12');
  assert.equal(edgeName(large, 'A11'), 'inside');
  assert.equal(edgeName(large, 'A12'), 'south edge');
  for (const label of ['M1', 'A13', 'A01']) {
    assert.throws(() => cellBounds(large, label), /Unknown cell/);
  }
  assert.throws(() => cellBounds(previewPlayground({ latitude: 0, longitude: 0 }, 1000, 3), 'D1'), /Unknown cell/);
  for (const cells of [2, 13, 6.5, NaN]) {
    assert.throws(() => previewPlayground({ latitude: 0, longitude: 0 }, 1000, cells), /grid/);
  }
});

test('whole-number settings accept only valid whole numbers', () => {
  assert.equal(parseWholeNumber(' 12 ', isValidGridSize), 12);
  assert.equal(parseWholeNumber('3', isValidGridSize), 3);
  assert.equal(parseWholeNumber('1', isValidRevealInterval), 1);
  assert.equal(parseWholeNumber('60', isValidRevealInterval), 60);
  for (const text of ['', '2', '13', '6.5', '6,0', '-6', 'abc', '1e1']) {
    assert.equal(parseWholeNumber(text, isValidGridSize), null, text);
  }
  for (const text of ['0', '61', '0.5']) {
    assert.equal(parseWholeNumber(text, isValidRevealInterval), null, text);
  }
});

test('default cells are 2 km and the playground is 12 km wide', () => {
  const p = previewPlayground({ latitude: 0, longitude: 0 });
  assert.equal(p.cellSizeMeters, 2000);
  assert.ok(Math.abs((p.bounds.north - p.bounds.south) * Math.PI / 180 * 6371008.8 - 12000) < 1e-6);
});

test('preview respects adjustable sizes at different latitudes', () => {
  for (const latitude of [-80, -52.52, 0, 52.52, 80]) {
    for (const size of [100, 1500, 2000, 5000]) {
      const p = previewPlayground({ latitude, longitude: 13.405 }, size);
      const height = (p.bounds.north - p.bounds.south) * Math.PI / 180 * 6371008.8;
      const width = (p.bounds.east - p.bounds.west) * Math.PI / 180 * 6371008.8 * Math.cos(latitude * Math.PI / 180);
      assert.ok(Math.abs(height - 6 * size) < 1e-6);
      assert.ok(Math.abs(width - 6 * size) < 1e-6);
      assert.equal(p.rows, 6);
      assert.equal(p.columns, 6);
      assert.equal(p.cellSizeMeters, size);
    }
  }
});

test('cell size input accepts comma/dot decimals and validates range and increments', () => {
  for (const [text, size] of [['2', 2000], ['2.0', 2000], ['1,5', 1500], ['1.50', 1500], [' 0,1 ', 100], ['5', 5000], ['1.1', 1100]] as const) {
    assert.equal(parseCellSizeKm(text), size);
  }
  for (const text of ['', '0', '-2', '0.05', '5.1', '1.55', 'Infinity', '2abc', '2e0', '1,2,3', '.']) {
    assert.equal(parseCellSizeKm(text), null);
  }
  for (const size of [0, 99, 1550, 5100, NaN, Infinity]) {
    assert.throws(() => previewPlayground({ latitude: 0, longitude: 0 }, size));
  }
});

test('grid has four corners, ten interior lines, and 36 correctly ordered labels', () => {
  const p = previewPlayground({ latitude: 52.52, longitude: 13.405 });
  assert.equal(boundaryCoordinates(p.bounds).length, 4);
  const lines = gridLines(p);
  assert.equal(lines.length, 10);
  for (const line of lines) {
    assert.equal(line.length, 2);
    for (const c of line) {
      assert.ok(c.latitude >= p.bounds.south && c.latitude <= p.bounds.north);
      assert.ok(c.longitude >= p.bounds.west && c.longitude <= p.bounds.east);
    }
  }
  const labels = cellLabels(p);
  assert.equal(labels.length, 36);
  assert.equal(new Set(labels.map((c) => c.label)).size, 36);
  assert.equal(labels[0].label, 'A1');
  assert.equal(labels[5].label, 'F1');
  assert.equal(labels[30].label, 'A6');
  assert.equal(labels[35].label, 'F6');
  assert.ok(labels[0].coordinate.latitude > labels[30].coordinate.latitude);
  assert.ok(labels[0].coordinate.longitude < labels[5].coordinate.longitude);
});

test('invalid centers and date-line crossings cannot be previewed', () => {
  for (const center of [
    { latitude: 81, longitude: 0 }, { latitude: 0, longitude: 181 },
    { latitude: NaN, longitude: 0 }, { latitude: 0, longitude: Infinity },
    { latitude: 0, longitude: 180 }, { latitude: 0, longitude: -180 },
  ]) assert.throws(() => previewPlayground(center));
});

test('a rotated grid keeps its cells, labels and edge rows', () => {
  for (const degrees of [1, 30, 45, 90, 180, 271, 359]) {
    const p = previewPlayground({ latitude: 52.52, longitude: 13.405 }, 1000, 4, degrees);
    assert.equal(p.rotationDegrees, degrees);
    for (const { label, coordinate } of cellLabels(p)) {
      assert.equal(cellAt(p, coordinate), label, `${degrees}°: ${label}`);
      assert.deepEqual(coordinate, cellCenter(p, label));
    }
    const centers = new Map(cellLabels(p).map(({ label, coordinate }) => [label, coordinate]));
    assert.equal(routePointAt(p, centers.get('B4')!), 'start');
    assert.equal(routePointAt(p, centers.get('D1')!), 'stop');
    assert.equal(routePointAt(p, centers.get('C2')!), null);
  }
});

test('a rotated grid turns clockwise around its center', () => {
  const p = previewPlayground({ latitude: 52.52, longitude: 13.405 }, 1000, 4, 90);
  // Turned by 90°, row 1 lies in the east and column A in the north.
  const a1 = cellCenter(p, 'A1');
  assert.ok(a1.latitude > p.center.latitude && a1.longitude > p.center.longitude);
  assert.equal(cellAt(p, { latitude: p.center.latitude, longitude: p.bounds.east - 0.001 })?.slice(1), '1');
  // The corners of the playground and of each cell turn with it and keep the center in the middle.
  const corners = playgroundCorners(p);
  assert.equal(corners.length, 4);
  const middle = corners.reduce((sum, c) => ({ latitude: sum.latitude + c.latitude / 4, longitude: sum.longitude + c.longitude / 4 }), { latitude: 0, longitude: 0 });
  assert.ok(Math.abs(middle.latitude - p.center.latitude) < 1e-12 && Math.abs(middle.longitude - p.center.longitude) < 1e-12);
  for (const corner of cellCorners(p, 'C3')) {
    assert.ok(Math.abs(corner.latitude - cellCenter(p, 'C3').latitude) < (p.bounds.north - p.bounds.south) / 4);
  }
});

test('a grid turned by 45° covers other ground than the north-aligned one', () => {
  const p = previewPlayground({ latitude: 52.52, longitude: 13.405 }, 1000, 4, 45);
  // The north-aligned corner lies outside; A1 sticks out beyond the north-aligned grid.
  assert.equal(cellAt(p, { latitude: p.bounds.north - 0.0001, longitude: p.bounds.west + 0.0001 }), null);
  const a1 = cellCenter(p, 'A1');
  assert.ok(a1.latitude > p.bounds.north);
  assert.equal(cellAt(p, a1), 'A1');
  // Grid lines and corners leave the north-aligned bounds.
  assert.ok(playgroundCorners(p).some((c) => c.latitude > p.bounds.north));
  assert.equal(gridLines(p).length, 6);
});

test('the rotation must be whole degrees from 0 to 359 and keep the grid off the date line', () => {
  for (const degrees of [-1, 360, 1.5, NaN]) {
    assert.throws(() => previewPlayground({ latitude: 0, longitude: 0 }, 1000, 4, degrees), /rotation/);
  }
  // 3 × 100 m: 150 m from the center to an edge, about 212 m to a corner.
  const longitude = 180 - 180 / Math.PI * 180 / 6371008.8;
  assert.doesNotThrow(() => previewPlayground({ latitude: 0, longitude }, 100, 3, 0));
  assert.throws(() => previewPlayground({ latitude: 0, longitude }, 100, 3, 45), /date line/);
});

test('distanceMeters is the haversine distance', () => {
  const berlin = { latitude: 52.52, longitude: 13.405 };
  const paris = { latitude: 48.8566, longitude: 2.3522 };
  assert.ok(Math.abs(distanceMeters(berlin, paris) - 877_000) < 3_000);
  assert.equal(distanceMeters(berlin, berlin), 0);
});
