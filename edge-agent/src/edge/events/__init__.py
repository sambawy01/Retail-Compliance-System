"""Event subjects mirroring the Go backend's retail compliance event constants.

DO NOT inline these strings anywhere else — always import. The cross-
language parity test asserts the Python set matches the Go set exactly.

16 retail event subjects grouped by severity:
  Critical (5): safety.slip_fall, theft.cash_drawer, access.after_hours,
                labor.buddy_punch, compliance.blocked_exit
  Warning  (7): compliance.uniform_violation, compliance.hygiene_violation,
                compliance.phone_usage, compliance.cleanliness_alert,
                operations.checkout_bottleneck, inventory.stockroom_anomaly,
                security.loitering, camera.degraded
  Info     (4): customer.loyalty_recognized, occupancy.update, activity.update
"""
from __future__ import annotations

# -- Critical severity ---------------------------------------------------
SAFETY_SLIP_FALL = "vision.safety.slip_fall"
THEFT_CASH_DRAWER = "vision.theft.cash_drawer"
ACCESS_AFTER_HOURS = "vision.access.after_hours"
LABOR_BUDDY_PUNCH = "vision.labor.buddy_punch"
COMPLIANCE_BLOCKED_EXIT = "vision.compliance.blocked_exit"

# -- Warning severity ----------------------------------------------------
COMPLIANCE_UNIFORM_VIOLATION = "vision.compliance.uniform_violation"
COMPLIANCE_HYGIENE_VIOLATION = "vision.compliance.hygiene_violation"
COMPLIANCE_PHONE_USAGE = "vision.compliance.phone_usage"
COMPLIANCE_CLEANLINESS_ALERT = "vision.compliance.cleanliness_alert"
OPS_CHECKOUT_BOTTLENECK = "vision.operations.checkout_bottleneck"
INVENTORY_STOCKROOM_ANOMALY = "vision.inventory.stockroom_anomaly"
SECURITY_LOITERING = "vision.security.loitering"
CAMERA_DEGRADED = "vision.camera.degraded"

# -- Info severity -------------------------------------------------------
CUSTOMER_LOYALTY_RECOGNIZED = "vision.customer.loyalty_recognized"
OCCUPANCY_UPDATE = "vision.occupancy.update"
ACTIVITY_UPDATE = "vision.activity.update"

ALL_SUBJECTS: dict[str, str] = {
    name: value
    for name, value in list(locals().items())
    if name.isupper() and isinstance(value, str) and value.startswith("vision.")
}