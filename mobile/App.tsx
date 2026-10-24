import { NavigationContainer } from '@react-navigation/native';
import { navigationTheme } from './src/theme/navigationTheme';
import { RootNavigator } from './src/navigation/RootNavigator';

export default function App() {
  return (
    <NavigationContainer theme={navigationTheme}>
      <RootNavigator />
    </NavigationContainer>
  );
}
