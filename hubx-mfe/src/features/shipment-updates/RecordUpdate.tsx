import { useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'

/** A registered user with the `ground partner` role (GET /api/ground-partners). */
export interface GroundPartner {
  id: string
  name: string
}

/** The five stages that can be recorded; `Created` is never an update stage. */
export const UPDATE_STAGES = [
  'In flight',
  'At destination airport',
  'Assigned to partner',
  'Out for delivery',
  'Delivered',
] as const

export type UpdateStage = (typeof UPDATE_STAGES)[number]

/** Request body of POST /api/shipments/{code}/updates. */
export interface RecordUpdateBody {
  stage: UpdateStage
  assignedPartnerId: string | null
}

/** Responsible party carried by a recorded update. */
export interface ResponsibleParty {
  id: string
  name: string
  role: 'operator' | 'ground partner' | 'customer'
}

/** 201 response of POST /api/shipments/{code}/updates. */
export interface RecordedUpdate {
  id: number
  shipmentCode: string
  sequence: number
  stage: UpdateStage
  responsibleParty: ResponsibleParty
  assignedPartner: GroundPartner | null
  recordedAt: string
  previousFingerprint: string | null
  fingerprint: string
}

/** Outcome returned by the injected transport; this component makes no HTTP call. */
export type RecordUpdateOutcome =
  | { kind: 'success'; update: RecordedUpdate }
  | { kind: 'httpError'; status: number; code: string; message: string }
  | { kind: 'noResponse' }

export interface RecordUpdateProps {
  groundPartners: readonly GroundPartner[]
  onRecordUpdate: (shipmentCode: string, body: RecordUpdateBody) => Promise<RecordUpdateOutcome>
  onSessionExpired: () => void
}

type SubmissionState =
  | { kind: 'idle' }
  | { kind: 'submitting' }
  | { kind: 'unexpectedError' }
  | RecordUpdateOutcome

function isUpdateStage(value: string): value is UpdateStage {
  return UPDATE_STAGES.some((stage) => stage === value)
}

function submissionMessage(state: SubmissionState): string {
  switch (state.kind) {
    case 'idle':
      return 'Enter the shipment code and stage, then record the update.'
    case 'submitting':
      return 'Recording update…'
    case 'success': {
      const update = state.update
      const partner = update.assignedPartner ? `; ground partner ${update.assignedPartner.name}` : ''
      return `Update recorded: ${update.stage} for ${update.shipmentCode}; responsible party ${update.responsibleParty.name}; recorded at ${update.recordedAt}${partner}.`
    }
    case 'httpError':
      return state.status === 401 || state.code === 'SESSION_EXPIRED'
        ? 'Session expired; sign in again.'
        : state.message
    case 'noResponse':
      return 'No response from the server. Try again.'
    case 'unexpectedError':
      return 'The update could not be recorded.'
  }
}

function RecordUpdate({ groundPartners, onRecordUpdate, onSessionExpired }: RecordUpdateProps) {
  const [shipmentCode, setShipmentCode] = useState('')
  const [stage, setStage] = useState<UpdateStage | null>(null)
  const [assignedPartnerId, setAssignedPartnerId] = useState('')
  const [submission, setSubmission] = useState<SubmissionState>({ kind: 'idle' })

  const needsPartner = stage === 'Assigned to partner'
  const busy = submission.kind === 'submitting'

  const handleCodeChange = (event: ChangeEvent<HTMLInputElement>) => {
    setShipmentCode(event.target.value)
    setSubmission({ kind: 'idle' })
  }

  const handleStageChange = (event: ChangeEvent<HTMLSelectElement>) => {
    const value = event.target.value
    setStage(value === '' || !isUpdateStage(value) ? null : value)
    if (value !== 'Assigned to partner') {
      setAssignedPartnerId('')
    }
    setSubmission({ kind: 'idle' })
  }

  const handlePartnerChange = (event: ChangeEvent<HTMLSelectElement>) => {
    setAssignedPartnerId(event.target.value)
    setSubmission({ kind: 'idle' })
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (busy) {
      return
    }

    const code = shipmentCode.trim()
    if (!code || !stage) {
      setSubmission({
        kind: 'httpError',
        status: 400,
        code: 'INVALID_REQUEST',
        message: 'The update data is invalid.',
      })
      return
    }
    if (needsPartner && !assignedPartnerId) {
      setSubmission({
        kind: 'httpError',
        status: 400,
        code: 'PARTNER_REQUIRED',
        message: 'You must name the ground partner responsible for the delivery.',
      })
      return
    }

    const body: RecordUpdateBody = {
      stage,
      assignedPartnerId: needsPartner ? assignedPartnerId : null,
    }
    setSubmission({ kind: 'submitting' })
    try {
      const outcome = await onRecordUpdate(code, body)
      setSubmission(outcome)
      if (outcome.kind === 'httpError' && (outcome.status === 401 || outcome.code === 'SESSION_EXPIRED')) {
        onSessionExpired()
      }
    } catch {
      setSubmission({ kind: 'unexpectedError' })
    }
  }

  const handleFormSubmit = (event: FormEvent<HTMLFormElement>) => {
    void handleSubmit(event)
  }

  return (
    <section aria-labelledby="record-update-heading">
      <h2 id="record-update-heading">Record an update</h2>
      <form noValidate onSubmit={handleFormSubmit}>
        <div>
          <label htmlFor="shipment-code">Shipment code</label>
          <input
            id="shipment-code"
            name="shipmentCode"
            type="text"
            value={shipmentCode}
            onChange={handleCodeChange}
            disabled={busy}
            required
          />
        </div>
        <div>
          <label htmlFor="update-stage">Stage</label>
          <select
            id="update-stage"
            name="stage"
            value={stage ?? ''}
            onChange={handleStageChange}
            disabled={busy}
            required
          >
            <option value="">Select a stage</option>
            {UPDATE_STAGES.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        </div>
        {needsPartner && (
          <div>
            <label htmlFor="ground-partner">Ground partner</label>
            <select
              id="ground-partner"
              name="assignedPartnerId"
              value={assignedPartnerId}
              onChange={handlePartnerChange}
              disabled={busy || groundPartners.length === 0}
              required
            >
              <option value="">Select a ground partner</option>
              {groundPartners.map((partner) => (
                <option key={partner.id} value={partner.id}>
                  {partner.name}
                </option>
              ))}
            </select>
            {groundPartners.length === 0 && <p>No ground partners available.</p>}
          </div>
        )}
        <button type="submit" disabled={busy}>
          Record update
        </button>
      </form>
      <p role="status" aria-live="polite">
        {submissionMessage(submission)}
      </p>
    </section>
  )
}

export default RecordUpdate
