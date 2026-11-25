/**
 * Create Shipment page for operators.
 * Allows operators to create a new shipment with origin, destination, flight, weight, and customer.
 */

import { useState, type FormEvent } from 'react';
import { AppShell } from '../components/AppShell';
import { useAuth } from '../contexts/AuthContext';
import { createShipment, type Shipment } from '../services/shipmentService';

export function CreateShipment() {
  const { user } = useAuth();

  const [origin, setOrigin] = useState('');
  const [destination, setDestination] = useState('');
  const [flight, setFlight] = useState('');
  const [weight, setWeight] = useState('');
  const [customerEmail, setCustomerEmail] = useState('');

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [createdShipment, setCreatedShipment] = useState<Shipment | null>(null);

  // Non-operators see permission message and no form
  if (user?.role !== 'operator') {
    return (
      <AppShell narrow showBack>
        <h1>Create Shipment</h1>
        <p className="message" data-tone="bad">
          You do not have permission to create shipments
        </p>
      </AppShell>
    );
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const shipment = await createShipment({
        origin: origin.trim(),
        destination: destination.trim(),
        flight: flight.trim(),
        weightKg: parseFloat(weight),
        customerEmail: customerEmail.trim(),
      });

      setCreatedShipment(shipment);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An unexpected error occurred');
    } finally {
      setLoading(false);
    }
  };

  // Success view
  if (createdShipment) {
    return (
      <AppShell narrow showBack>
        <h1>Shipment created</h1>
        <div className="panel form">
          <div className="message" data-tone="ok">
            <p className="shipment-code">Code: {createdShipment.code}</p>
            <p>Stage: {createdShipment.stage}</p>
          </div>
          <dl className="details">
            <dt>Origin</dt>
            <dd>{createdShipment.origin}</dd>
            <dt>Destination</dt>
            <dd>{createdShipment.destination}</dd>
            <dt>Flight</dt>
            <dd>{createdShipment.flight}</dd>
            <dt>Weight</dt>
            <dd>{createdShipment.weightKg} kg</dd>
            <dt>Customer</dt>
            <dd>
              {createdShipment.customer.name} ({createdShipment.customer.email})
            </dd>
          </dl>
        </div>
      </AppShell>
    );
  }

  // Form view
  return (
    <AppShell narrow showBack>
      <h1>Create Shipment</h1>

      {error && (
        <div className="message" data-tone="bad" role="alert">
          {error}
        </div>
      )}

      <form className="panel form" onSubmit={handleSubmit}>
        <div className="field-pair">
          <div className="field">
            <label htmlFor="origin">
              Origin
            </label>
            <input
              id="origin"
              type="text"
              value={origin}
              onChange={(e) => setOrigin(e.target.value)}
              disabled={loading}
              required
            />
          </div>

          <div className="field">
            <label htmlFor="destination">
              Destination
            </label>
            <input
              id="destination"
              type="text"
              value={destination}
              onChange={(e) => setDestination(e.target.value)}
              disabled={loading}
              required
            />
          </div>
        </div>

        <div className="field-pair">
          <div className="field">
            <label htmlFor="flight">
              Flight
            </label>
            <input
              id="flight"
              type="text"
              value={flight}
              onChange={(e) => setFlight(e.target.value)}
              disabled={loading}
              required
            />
          </div>

          <div className="field">
            <label htmlFor="weight">
              Weight (kg)
            </label>
            <input
              id="weight"
              type="number"
              step="0.1"
              min="0.1"
              max="5000"
              value={weight}
              onChange={(e) => setWeight(e.target.value)}
              disabled={loading}
              required
            />
          </div>
        </div>

        <div className="field">
          <label htmlFor="customerEmail">
            Customer email
          </label>
          <input
            id="customerEmail"
            type="email"
            value={customerEmail}
            onChange={(e) => setCustomerEmail(e.target.value)}
            disabled={loading}
            required
          />
        </div>

        <button type="submit" className="button" disabled={loading}>
          {loading ? 'Creating...' : 'Create shipment'}
        </button>
      </form>
    </AppShell>
  );
}
