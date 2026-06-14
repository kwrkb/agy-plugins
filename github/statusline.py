#!/usr/bin/env python3
import sys
import json

def main():
    try:
        data = json.load(sys.stdin)
    except Exception:
        print("● ERROR | Statusline parser failed")
        return

    # Extract state, model, sandbox, cwd, context window
    state = data.get("agent_state", "idle").upper()
    model_name = data.get("model", {}).get("display_name", "Unknown Model")
    
    sandbox_enabled = data.get("sandbox", {}).get("enabled", False)
    sandbox_str = "🛡️" if sandbox_enabled else "⚠️"
    
    cwd = data.get("cwd", "")
    folder_name = cwd.split("/")[-1] if cwd else "unknown"
    
    used_pct = data.get("context_window", {}).get("used_percentage", 0.0)
    
    # Extract 5h quotas
    quota = data.get("quota", {})
    quota_parts = []
    
    gemini_5h = quota.get("gemini-5h")
    if gemini_5h:
        gemini_pct = int(gemini_5h.get("remaining_fraction", 1.0) * 100)
        quota_parts.append(f"5h(Gemini): {gemini_pct}%")
        
    tp_5h = quota.get("3p-5h")
    if tp_5h:
        tp_pct = int(tp_5h.get("remaining_fraction", 1.0) * 100)
        quota_parts.append(f"5h(3P): {tp_pct}%")
        
    quota_str = " | ".join(quota_parts) if quota_parts else "5h: N/A"

    # Status icon
    status_icon = "●"
    if state == "WORKING":
        status_icon = "⚙️"

    # Build statusline output
    output = f"{status_icon} {state} | [{model_name}] {sandbox_str} 📁 {folder_name} | {used_pct:.1f}% ctx | {quota_str}"
    print(output)

if __name__ == "__main__":
    main()
