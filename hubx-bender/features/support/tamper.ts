import { spawn } from 'node:child_process';

/**
 * Tamper an update in the isolated test database by directly altering its stage field.
 * This bypasses the immutability trigger on purpose (SET LOCAL session_replication_role = replica).
 * 
 * Only used in test scenarios tagged @needs-secret; deployments never run it.
 * 
 * @param shipmentCode - The shipment code (validated against ^HBX-[0-9A-F]{8}$)
 * @param sequence - The update's sequence number (1-based)
 * @param newStage - The tampered stage value
 */
export async function tamperUpdate(shipmentCode: string, sequence: number, newStage: string): Promise<void> {
  // Validate shipment code format to prevent SQL injection.
  if (!/^HBX-[0-9A-F]{8}$/.test(shipmentCode)) {
    throw new Error(`Invalid shipment code format: ${shipmentCode}`);
  }
  
  // Restrict newStage to known stage values to prevent SQL injection.
  const validStages = ['In flight', 'At destination airport', 'Assigned to partner', 'Out for delivery', 'Delivered'];
  if (!validStages.includes(newStage)) {
    throw new Error(`Invalid stage value: ${newStage}`);
  }
  
  // Compose project name from environment (defaults to hubx-it, the isolated test stack).
  const composeProject = process.env.BENDER_COMPOSE_PROJECT ?? 'hubx-it';
  
  // PostgreSQL connection details from compose.yaml (service postgres, POSTGRES_DB, POSTGRES_USER).
  const dbName = 'hubx';
  const dbUser = 'hubx';
  
  // SQL to tamper the update: bypass the immutability trigger and alter the stage field.
  // The shipment code and sequence are passed as psql variables; psql interpolates them from stdin.
  const sql = `
BEGIN;
SET LOCAL session_replication_role = replica;
UPDATE updates 
SET stage = :'newStage'
WHERE sequence = ${sequence}
  AND shipment_id = (SELECT id FROM shipments WHERE code = :'code');
COMMIT;
`.trim();
  
  return new Promise<void>((resolve, reject) => {
    // Run docker compose exec -T postgres psql with SQL piped to stdin.
    // No shell; shipment code and stage passed as psql variables.
    const child = spawn(
      'docker',
      [
        'compose',
        '-p', composeProject,
        'exec', '-T', 'postgres',
        'psql',
        '-U', dbUser,
        '-d', dbName,
        '-v', `code=${shipmentCode}`,
        '-v', `newStage=${newStage}`,
        '-v', 'ON_ERROR_STOP=1',
        '-f', '-',
      ],
      {
        stdio: ['pipe', 'pipe', 'pipe'],
        timeout: 5000,
      }
    );
    
    let stdout = '';
    let stderr = '';
    
    child.stdout.on('data', (data) => {
      stdout += data.toString();
    });
    
    child.stderr.on('data', (data) => {
      stderr += data.toString();
    });
    
    child.on('error', (error) => {
      reject(new Error(`Failed to spawn docker: ${error.message}`));
    });
    
    child.on('close', (code) => {
      if (code !== 0) {
        reject(new Error(
          `Failed to tamper update ${sequence} of ${shipmentCode}: exit ${code}\n${stderr || stdout}`
        ));
        return;
      }
      
      // Log the tampering for debugging (not shown in Cucumber output unless the scenario fails).
      if (stderr && !stderr.includes('NOTICE')) {
        console.warn(`Tamper stderr: ${stderr}`);
      }
      console.log(`Tampered update ${sequence} of ${shipmentCode} to stage "${newStage}"`);
      resolve();
    });
    
    // Write SQL to stdin and close the stream.
    child.stdin.write(sql);
    child.stdin.end();
  });
}
