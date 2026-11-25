/**
 * Visual summary of the stages a shipment went through. Hidden from assistive
 * technology: the current stage and the updates list already state it in text.
 */

import { UPDATE_STAGES } from '../shipment-updates/RecordUpdate';

const STAGES = ['Created', ...UPDATE_STAGES];

interface StageTrackProps {
  currentStage: string;
  /** Stages that have an update; a stage without one is never shown as recorded. */
  recordedStages: readonly string[];
}

export function StageTrack({ currentStage, recordedStages }: StageTrackProps) {
  const stateOf = (stage: string): 'current' | 'recorded' | 'pending' => {
    if (stage === currentStage) {
      return 'current';
    }
    return stage === 'Created' || recordedStages.includes(stage) ? 'recorded' : 'pending';
  };

  return (
    <div className="track" aria-hidden="true">
      {STAGES.map((stage) => (
        <div key={stage} className="track-stop" data-state={stateOf(stage)}>
          {stage}
        </div>
      ))}
    </div>
  );
}
