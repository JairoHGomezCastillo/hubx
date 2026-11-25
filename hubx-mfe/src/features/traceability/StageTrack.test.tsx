import { describe, expect, it } from 'vitest';
import { render } from '@testing-library/react';
import { StageTrack } from './StageTrack';

function states(container: HTMLElement) {
  return [...container.querySelectorAll<HTMLElement>('[data-state]')].map((stop) => stop.dataset.state);
}

describe('StageTrack', () => {
  it('marks Created as current when there are no updates', () => {
    const { container } = render(<StageTrack currentStage="Created" recordedStages={[]} />);
    expect(states(container)).toEqual(['current', 'pending', 'pending', 'pending', 'pending', 'pending']);
  });

  it('never shows a stage without an update as recorded', () => {
    const { container } = render(
      <StageTrack currentStage="Assigned to partner" recordedStages={['In flight', 'Assigned to partner']} />,
    );
    expect(states(container)).toEqual(['recorded', 'recorded', 'pending', 'current', 'pending', 'pending']);
  });
});
