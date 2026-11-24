/**
 * Create Shipment page for operators.
 * Allows operators to create a new shipment with origin, destination, flight, weight, and customer.
 */

import { useState, type FormEvent } from 'react';
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
      <main style={{ maxWidth: '800px', margin: '40px auto', padding: '20px' }}>
        <h1>Create Shipment</h1>
        <p style={{ color: '#dc3545', marginTop: '20px' }}>
          You do not have permission to create shipments
        </p>
      </main>
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
      <main style={{ maxWidth: '800px', margin: '40px auto', padding: '20px' }}>
        <h1>Shipment created</h1>
        <div
          style={{
            marginTop: '24px',
            padding: '20px',
            backgroundColor: '#d4edda',
            border: '1px solid #c3e6cb',
            borderRadius: '4px',
          }}
        >
          <p style={{ margin: '0 0 12px 0', fontSize: '18px' }}>
            Code: {createdShipment.code}
          </p>
          <p style={{ margin: '0', fontSize: '18px' }}>
            Stage: {createdShipment.stage}
          </p>
        </div>
        <div style={{ marginTop: '24px' }}>
          <p>
            <strong>Origin:</strong> {createdShipment.origin}
          </p>
          <p>
            <strong>Destination:</strong> {createdShipment.destination}
          </p>
          <p>
            <strong>Flight:</strong> {createdShipment.flight}
          </p>
          <p>
            <strong>Weight:</strong> {createdShipment.weightKg} kg
          </p>
          <p>
            <strong>Customer:</strong> {createdShipment.customer.name} ({createdShipment.customer.email})
          </p>
        </div>
      </main>
    );
  }

  // Form view
  return (
    <main style={{ maxWidth: '800px', margin: '40px auto', padding: '20px' }}>
      <h1>Create Shipment</h1>

      {error && (
        <div
          role="alert"
          style={{
            marginTop: '20px',
            padding: '12px 16px',
            backgroundColor: '#f8d7da',
            border: '1px solid #f5c6cb',
            borderRadius: '4px',
            color: '#721c24',
          }}
        >
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} style={{ marginTop: '24px' }}>
        <div style={{ marginBottom: '16px' }}>
          <label
            htmlFor="origin"
            style={{
              display: 'block',
              marginBottom: '8px',
              fontWeight: '500',
            }}
          >
            Origin
          </label>
          <input
            id="origin"
            type="text"
            value={origin}
            onChange={(e) => setOrigin(e.target.value)}
            disabled={loading}
            required
            style={{
              width: '100%',
              padding: '8px 12px',
              fontSize: '16px',
              border: '1px solid #ced4da',
              borderRadius: '4px',
              boxSizing: 'border-box',
            }}
          />
        </div>

        <div style={{ marginBottom: '16px' }}>
          <label
            htmlFor="destination"
            style={{
              display: 'block',
              marginBottom: '8px',
              fontWeight: '500',
            }}
          >
            Destination
          </label>
          <input
            id="destination"
            type="text"
            value={destination}
            onChange={(e) => setDestination(e.target.value)}
            disabled={loading}
            required
            style={{
              width: '100%',
              padding: '8px 12px',
              fontSize: '16px',
              border: '1px solid #ced4da',
              borderRadius: '4px',
              boxSizing: 'border-box',
            }}
          />
        </div>

        <div style={{ marginBottom: '16px' }}>
          <label
            htmlFor="flight"
            style={{
              display: 'block',
              marginBottom: '8px',
              fontWeight: '500',
            }}
          >
            Flight
          </label>
          <input
            id="flight"
            type="text"
            value={flight}
            onChange={(e) => setFlight(e.target.value)}
            disabled={loading}
            required
            style={{
              width: '100%',
              padding: '8px 12px',
              fontSize: '16px',
              border: '1px solid #ced4da',
              borderRadius: '4px',
              boxSizing: 'border-box',
            }}
          />
        </div>

        <div style={{ marginBottom: '16px' }}>
          <label
            htmlFor="weight"
            style={{
              display: 'block',
              marginBottom: '8px',
              fontWeight: '500',
            }}
          >
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
            style={{
              width: '100%',
              padding: '8px 12px',
              fontSize: '16px',
              border: '1px solid #ced4da',
              borderRadius: '4px',
              boxSizing: 'border-box',
            }}
          />
        </div>

        <div style={{ marginBottom: '24px' }}>
          <label
            htmlFor="customerEmail"
            style={{
              display: 'block',
              marginBottom: '8px',
              fontWeight: '500',
            }}
          >
            Customer email
          </label>
          <input
            id="customerEmail"
            type="email"
            value={customerEmail}
            onChange={(e) => setCustomerEmail(e.target.value)}
            disabled={loading}
            required
            style={{
              width: '100%',
              padding: '8px 12px',
              fontSize: '16px',
              border: '1px solid #ced4da',
              borderRadius: '4px',
              boxSizing: 'border-box',
            }}
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          style={{
            padding: '12px 24px',
            fontSize: '16px',
            backgroundColor: loading ? '#6c757d' : '#007bff',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: loading ? 'not-allowed' : 'pointer',
          }}
        >
          {loading ? 'Creating...' : 'Create shipment'}
        </button>
      </form>
    </main>
  );
}
