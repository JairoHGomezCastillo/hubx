/**
 * Wires ViewTraceability to the HU-04 /api client.
 * Kept separate from the presentational component so that every query state is
 * explicit and the component can be tested with an injected transport.
 */

import { getTraceability } from '../../api/client';
import ViewTraceability from './ViewTraceability';

interface ViewTraceabilityContainerProps {
  onSessionExpired: () => void;
}

export function ViewTraceabilityContainer({ onSessionExpired }: ViewTraceabilityContainerProps) {
  return <ViewTraceability onViewTraceability={getTraceability} onSessionExpired={onSessionExpired} />;
}
