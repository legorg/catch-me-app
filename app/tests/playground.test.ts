import assert from 'node:assert/strict';
import { test } from 'node:test';

import {
  boundaryCoordinates, cellBounds, cellLabels, edgeName, gridLines, parseCellSizeKm, previewPlayground,
} from '../src/playground';

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
  assert.equal(edgeName('B1'), 'north edge');
  assert.equal(edgeName('E6'), 'south edge');
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
