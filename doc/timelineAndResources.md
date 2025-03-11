```mermaid
gantt
    title TC2Go Project Timeline and Resources
    dateFormat  YYYY-MM-DD
    axisFormat  WW

    section WW11
        Basic Int Test (BLR)              :a1, 2025-03-10, 1w
        UNIX Socket (Gavin)               :done, a2, 2025-03-10, 1w
        Repo Struct (Gavin)               :done, a3, 2025-03-10, 1w
        INBC Command-line tool (Nat)      :done, a4, 2025-03-10, 1w
        Create Initial Earthfile (Nat)    :active, a5, 2025-03-10, 1w
        TC Systemd service (YL)           :active, a6, 2025-03-10, 1w        

    section WW12
        CI/CD Int (BLR)                   :b1, 2025-03-17, 2w
        SOTA Download-only Tiber (YL)     :crit, b2, 2025-03-17, 1w
        Unit Tests (BLR)                  :a6, 2025-03-10, 3w

    section WW13
        SOTA Update Tiber (YL)            :crit, c1, 2025-03-24, 2w
        Demo Prep (Gavin/YL)              :crit, c2, 2025-03-24, 2w

    section WW14
        Fuzz Testing (Val)                :d1, 2025-03-31, 6w

    section WW15
        ITEP Integration                  :e1, 2025-04-7, 2w
