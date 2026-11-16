import * as Location from 'expo-location';
import { useState, type RefObject } from 'react';
import type MapView from 'react-native-maps';
import type { LatLng } from 'react-native-maps';

import { useT } from './i18n';

/** The "my location" button: asks for permission, then puts the grid (editable) or just the camera on the user. */
export function useMapLocate(map: RefObject<MapView | null>, editable: boolean, select: (coordinate: LatLng) => void) {
  const t = useT();
  const [permission, requestPermission] = Location.useForegroundPermissions();
  const [locating, setLocating] = useState(false);
  const [locateError, setLocateError] = useState('');
  const locate = async () => {
    if (locating) return;
    setLocating(true);
    setLocateError('');
    try {
      const granted = permission?.granted || (await requestPermission()).granted;
      if (!granted) {
        setLocateError(t('map.locationDenied'));
        return;
      }
      const { coords } = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      const here = { latitude: coords.latitude, longitude: coords.longitude };
      // An editable map puts the grid there; any other map just shows where you are.
      if (editable) select(here);
      else map.current?.animateCamera({ center: here }, { duration: 300 });
    } catch {
      setLocateError(t('map.locationFailed'));
    } finally {
      setLocating(false);
    }
  };
  return { locationGranted: permission?.granted === true, locating, locateError, locate: () => void locate() };
}
