/**
 * Container for VerifyIntegrity that wires the service layer.
 */

import VerifyIntegrity from './VerifyIntegrity';
import type { VerifyIntegrityOutcome } from './VerifyIntegrity';
import { verifyIntegrity } from '../../services/integrityService';

export interface VerifyIntegrityContainerProps {
  onSessionExpired: () => void;
}

export function VerifyIntegrityContainer({ onSessionExpired }: VerifyIntegrityContainerProps) {
  const handleVerify = async (shipmentCode: string): Promise<VerifyIntegrityOutcome> => {
    try {
      const result = await verifyIntegrity(shipmentCode);
      return { kind: 'success', result };
    } catch (error) {
      if (error instanceof Error) {
        // Extract error details if available
        const message = error.message;
        
        // Check if this is a 404 error
        if (message.includes('No shipment exists with that code')) {
          return {
            kind: 'httpError',
            status: 404,
            code: 'SHIPMENT_NOT_FOUND',
            message,
          };
        }
        
        // Check if this is a 401 error
        if (message.includes('Unauthorized') || message.includes('Session expired')) {
          return {
            kind: 'httpError',
            status: 401,
            code: 'UNAUTHORIZED',
            message,
          };
        }
        
        // Other HTTP errors
        return {
          kind: 'httpError',
          status: 500,
          code: 'INTERNAL_SERVER_ERROR',
          message,
        };
      }
      return { kind: 'noResponse' };
    }
  };

  return <VerifyIntegrity onVerify={handleVerify} onSessionExpired={onSessionExpired} />;
}
