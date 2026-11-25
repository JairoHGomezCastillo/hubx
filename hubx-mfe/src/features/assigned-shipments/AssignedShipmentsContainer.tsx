/**
 * Container for AssignedShipments: wires the component to the shipmentService.
 */

import { listAssignedShipments } from '../../services/shipmentService';
import AssignedShipments from './AssignedShipments';

interface AssignedShipmentsContainerProps {
  onSessionExpired: () => void;
}

export function AssignedShipmentsContainer({ onSessionExpired }: AssignedShipmentsContainerProps) {
  return <AssignedShipments onLoadAssignedShipments={listAssignedShipments} onSessionExpired={onSessionExpired} />;
}
