/**
 * Wires StageCounts to the stageCountService.
 * Kept separate from the presentational component so that every query state is
 * explicit and the component can be tested with an injected transport.
 */

import { getStageCounts } from '../../services/stageCountService';
import StageCounts from './StageCounts';

interface StageCountsContainerProps {
  onSessionExpired: () => void;
}

export function StageCountsContainer({ onSessionExpired }: StageCountsContainerProps) {
  return <StageCounts onGetStageCounts={getStageCounts} onSessionExpired={onSessionExpired} />;
}
