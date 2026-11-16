import { StyleSheet, View, type ViewProps } from 'react-native';

type Props = Pick<ViewProps, 'onStartShouldSetResponder' | 'onMoveShouldSetResponder' | 'onResponderGrant' | 'onResponderMove' | 'onResponderRelease' | 'onResponderTerminate'>;

/** Over the map, under the location button: catches the finger while drawing. */
export default function MapDrawingOverlay(responder: Props) {
  return <View style={StyleSheet.absoluteFill} {...responder} />;
}
