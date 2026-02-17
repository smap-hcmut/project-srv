#!/usr/bin/env python3
"""
Comprehensive Crisis Config API Test Script.
Tests 5 valid creation cases + validation failure cases.
Uses http.client (no external dependencies).
"""
import http.client
import json
import sys
import time

BASE_HOST = "localhost"
BASE_PORT = 8080
API_PREFIX = "/api/v1"
TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImRhbmdxdW9jcGhvbmcxNzAzQGdtYWlsLmNvbSIsInJvbGUiOiJWSUVXRVIiLCJpc3MiOiJzbWFwLWF1dGgtc2VydmljZSIsInN1YiI6IjMzZmZjZmQ5LWY3ODItNDlhOS1hOGE2LTM1NjNiYzAyNzUwZCIsImF1ZCI6WyJzbWFwLWFwaSJdLCJleHAiOjE3NzEzNzgzOTgsImlhdCI6MTc3MTM0OTU5OCwianRpIjoiMTViYzc3N2EtNDlkMy00YTNmLWJiNmYtYmNiNTNjZDk5NjllIn0.27YdT4R-fr2j8y7M1fvhQR4OZqtk7IlE6Craq6ti0bU"

passed = 0
failed = 0

def req(method, path, body=None, expect_code=200, description=""):
    """Make an HTTP request and check the response code."""
    global passed, failed
    conn = http.client.HTTPConnection(BASE_HOST, BASE_PORT, timeout=10)
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {TOKEN}",
    }
    payload = json.dumps(body) if body else None
    try:
        conn.request(method, f"{API_PREFIX}{path}", body=payload, headers=headers)
        resp = conn.getresponse()
        data = resp.read().decode()
        status = resp.status
        if status == expect_code:
            passed += 1
            print(f"  ✅ [{status}] {description}")
        else:
            failed += 1
            print(f"  ❌ [{status}] {description} (expected {expect_code})")
            print(f"     Response: {data[:300]}")
        conn.close()
        try:
            return json.loads(data)
        except:
            return {}
    except Exception as e:
        failed += 1
        print(f"  ❌ [ERR] {description}: {e}")
        return {}


def create_campaign(name):
    """Helper to create a campaign and return its ID."""
    resp = req("POST", "/campaigns", {"name": name}, 200, f"Create Campaign: {name}")
    return resp.get("data", {}).get("campaign", {}).get("id", "")


def create_project(campaign_id, name, entity_type="product", entity_name="Test Entity"):
    """Helper to create a project and return its ID."""
    resp = req("POST", f"/campaigns/{campaign_id}/projects", {
        "name": name,
        "entity_type": entity_type,
        "entity_name": entity_name,
    }, 200, f"Create Project: {name}")
    return resp.get("data", {}).get("project", {}).get("id", "")


# ============================================================
# MAIN
# ============================================================
print("=" * 60)
print("CRISIS CONFIG COMPREHENSIVE TEST")
print("=" * 60)

# --- Setup: create campaign + 5 projects ---
print("\n--- SETUP: Create Campaign & Projects ---")
campaign_id = create_campaign(f"CrisisTest-{int(time.time())}")
if not campaign_id:
    print("FATAL: Failed to create campaign. Aborting.")
    sys.exit(1)

project_ids = []
for i in range(1, 6):
    pid = create_project(campaign_id, f"CrisisProject-{i}")
    if not pid:
        print(f"FATAL: Failed to create project {i}. Aborting.")
        sys.exit(1)
    project_ids.append(pid)

# --- Valid Cases: 5 different crisis configs ---
print("\n--- VALID CASES: 5 Crisis Configs ---")

# Case 1: Keywords only
print("\n  [Case 1] Keywords Trigger Only")
req("PUT", f"/projects/{project_ids[0]}/crisis-config", {
    "keywords_trigger": {
        "enabled": True,
        "logic": "OR",
        "groups": [
            {"name": "Brand Crisis", "keywords": ["scandal", "lawsuit", "fraud"], "weight": 10},
            {"name": "Product Issues", "keywords": ["recall", "defect", "danger"], "weight": 8},
        ]
    }
}, 200, "UPSERT: Keywords only")
req("GET", f"/projects/{project_ids[0]}/crisis-config", None, 200, "GET: Verify keywords config")

# Case 2: Volume only
print("\n  [Case 2] Volume Trigger Only")
req("PUT", f"/projects/{project_ids[1]}/crisis-config", {
    "volume_trigger": {
        "enabled": True,
        "metric": "post_count",
        "rules": [
            {"level": "WARNING", "threshold_percent_growth": 200, "comparison_window_hours": 6, "baseline": "avg_7d"},
            {"level": "CRITICAL", "threshold_percent_growth": 500, "comparison_window_hours": 1, "baseline": "avg_24h"},
        ]
    }
}, 200, "UPSERT: Volume only")
req("GET", f"/projects/{project_ids[1]}/crisis-config", None, 200, "GET: Verify volume config")

# Case 3: Sentiment only
print("\n  [Case 3] Sentiment Trigger Only")
req("PUT", f"/projects/{project_ids[2]}/crisis-config", {
    "sentiment_trigger": {
        "enabled": True,
        "min_sample_size": 50,
        "rules": [
            {"type": "negative_ratio", "threshold_percent": 40.0},
            {"type": "absa_aspect_alert", "critical_aspects": ["quality", "safety"], "negative_threshold_percent": 60.0},
        ]
    }
}, 200, "UPSERT: Sentiment only")
req("GET", f"/projects/{project_ids[2]}/crisis-config", None, 200, "GET: Verify sentiment config")

# Case 4: Influencer only
print("\n  [Case 4] Influencer Trigger Only")
req("PUT", f"/projects/{project_ids[3]}/crisis-config", {
    "influencer_trigger": {
        "enabled": True,
        "logic": "AND",
        "rules": [
            {"type": "macro_influencer", "min_followers": 100000, "required_sentiment": "negative"},
            {"type": "viral_post", "min_shares": 5000, "min_comments": 1000},
        ]
    }
}, 200, "UPSERT: Influencer only")
req("GET", f"/projects/{project_ids[3]}/crisis-config", None, 200, "GET: Verify influencer config")

# Case 5: All 4 triggers combined
print("\n  [Case 5] All 4 Triggers Combined")
req("PUT", f"/projects/{project_ids[4]}/crisis-config", {
    "keywords_trigger": {
        "enabled": True,
        "logic": "AND",
        "groups": [{"name": "All-in-one", "keywords": ["crisis", "emergency"], "weight": 5}]
    },
    "volume_trigger": {
        "enabled": True,
        "metric": "mention_count",
        "rules": [{"level": "WARNING", "threshold_percent_growth": 150, "comparison_window_hours": 12, "baseline": "avg_30d"}]
    },
    "sentiment_trigger": {
        "enabled": True,
        "min_sample_size": 100,
        "rules": [{"type": "negative_ratio", "threshold_percent": 30.0}]
    },
    "influencer_trigger": {
        "enabled": True,
        "logic": "OR",
        "rules": [{"type": "macro_influencer", "min_followers": 50000, "required_sentiment": "negative"}]
    }
}, 200, "UPSERT: All 4 triggers")
req("GET", f"/projects/{project_ids[4]}/crisis-config", None, 200, "GET: Verify all-4 config")

# --- Validation Failure Cases ---
print("\n--- VALIDATION FAILURE CASES (expect 400) ---")

# F1: Empty body (no triggers)
print("\n  [F1] Empty body — no triggers")
req("PUT", f"/projects/{project_ids[0]}/crisis-config", {}, 400, "FAIL: empty body")

# F2: Keywords enabled but no groups
print("\n  [F2] Keywords enabled, no groups")
req("PUT", f"/projects/{project_ids[0]}/crisis-config", {
    "keywords_trigger": {"enabled": True, "logic": "OR", "groups": []}
}, 400, "FAIL: keywords enabled, empty groups")

# F3: Keyword group missing name
print("\n  [F3] Keyword group missing name")
req("PUT", f"/projects/{project_ids[0]}/crisis-config", {
    "keywords_trigger": {"enabled": True, "logic": "OR", "groups": [
        {"name": "", "keywords": ["test"], "weight": 1}
    ]}
}, 400, "FAIL: keyword group empty name")

# F4: Keyword group missing keywords
print("\n  [F4] Keyword group empty keywords list")
req("PUT", f"/projects/{project_ids[0]}/crisis-config", {
    "keywords_trigger": {"enabled": True, "logic": "OR", "groups": [
        {"name": "Test", "keywords": [], "weight": 1}
    ]}
}, 400, "FAIL: keyword group empty keywords")

# F5: Keyword group weight <= 0
print("\n  [F5] Keyword group weight = 0")
req("PUT", f"/projects/{project_ids[0]}/crisis-config", {
    "keywords_trigger": {"enabled": True, "logic": "OR", "groups": [
        {"name": "Test", "keywords": ["abc"], "weight": 0}
    ]}
}, 400, "FAIL: keyword group weight=0")

# F6: Volume enabled but no rules
print("\n  [F6] Volume enabled, no rules")
req("PUT", f"/projects/{project_ids[0]}/crisis-config", {
    "volume_trigger": {"enabled": True, "metric": "post_count", "rules": []}
}, 400, "FAIL: volume enabled, empty rules")

# F7: Volume rule missing level
print("\n  [F7] Volume rule missing level")
req("PUT", f"/projects/{project_ids[0]}/crisis-config", {
    "volume_trigger": {"enabled": True, "metric": "post_count", "rules": [
        {"level": "", "threshold_percent_growth": 100, "comparison_window_hours": 1, "baseline": "avg_24h"}
    ]}
}, 400, "FAIL: volume rule empty level")

# F8: Volume rule threshold <= 0
print("\n  [F8] Volume rule threshold = 0")
req("PUT", f"/projects/{project_ids[0]}/crisis-config", {
    "volume_trigger": {"enabled": True, "metric": "post_count", "rules": [
        {"level": "WARNING", "threshold_percent_growth": 0, "comparison_window_hours": 1, "baseline": "avg_24h"}
    ]}
}, 400, "FAIL: volume rule threshold=0")

# F9: Sentiment enabled but no rules
print("\n  [F9] Sentiment enabled, no rules")
req("PUT", f"/projects/{project_ids[0]}/crisis-config", {
    "sentiment_trigger": {"enabled": True, "min_sample_size": 50, "rules": []}
}, 400, "FAIL: sentiment enabled, empty rules")

# F10: Sentiment rule missing type
print("\n  [F10] Sentiment rule missing type")
req("PUT", f"/projects/{project_ids[0]}/crisis-config", {
    "sentiment_trigger": {"enabled": True, "min_sample_size": 50, "rules": [
        {"type": "", "threshold_percent": 40}
    ]}
}, 400, "FAIL: sentiment rule empty type")

# F11: Influencer enabled but no rules
print("\n  [F11] Influencer enabled, no rules")
req("PUT", f"/projects/{project_ids[0]}/crisis-config", {
    "influencer_trigger": {"enabled": True, "logic": "OR", "rules": []}
}, 400, "FAIL: influencer enabled, empty rules")

# F12: Influencer rule missing type
print("\n  [F12] Influencer rule missing type")
req("PUT", f"/projects/{project_ids[0]}/crisis-config", {
    "influencer_trigger": {"enabled": True, "logic": "OR", "rules": [
        {"type": "", "min_followers": 100}
    ]}
}, 400, "FAIL: influencer rule empty type")

# --- Campaign Validation Tests ---
print("\n--- CAMPAIGN VALIDATION TESTS (expect 400) ---")

# CF1: Campaign create with empty name
print("\n  [CF1] Campaign create — empty name")
req("POST", "/campaigns", {"name": ""}, 400, "FAIL: campaign empty name")

# CF2: Campaign create — bad date format
print("\n  [CF2] Campaign create — bad date format")
req("POST", "/campaigns", {"name": "BadDate", "start_date": "2026-01-01", "end_date": "not-a-date"}, 400, "FAIL: campaign bad date format")

# CF3: Campaign create — end before start
print("\n  [CF3] Campaign create — end_date < start_date")
req("POST", "/campaigns", {
    "name": "BadRange",
    "start_date": "2026-12-01T00:00:00Z",
    "end_date": "2026-01-01T00:00:00Z"
}, 400, "FAIL: campaign end before start")

# CF4: Campaign update — invalid status
print("\n  [CF4] Campaign update — invalid status")
req("PUT", f"/campaigns/{campaign_id}", {"status": "BANANA"}, 400, "FAIL: campaign invalid status")

# --- Project Validation Tests ---
print("\n--- PROJECT VALIDATION TESTS (expect 400) ---")

# PF1: Project create — empty name
print("\n  [PF1] Project create — empty name")
req("POST", f"/campaigns/{campaign_id}/projects", {
    "name": "", "entity_type": "product", "entity_name": "Test"
}, 400, "FAIL: project empty name")

# PF2: Project create — invalid entity_type
print("\n  [PF2] Project create — invalid entity_type")
req("POST", f"/campaigns/{campaign_id}/projects", {
    "name": "Test", "entity_type": "banana", "entity_name": "Test"
}, 400, "FAIL: project invalid entity_type")

# PF3: Project create — empty entity_name
print("\n  [PF3] Project create — empty entity_name")
req("POST", f"/campaigns/{campaign_id}/projects", {
    "name": "Test", "entity_type": "product", "entity_name": ""
}, 400, "FAIL: project empty entity_name")

# PF4: Project create — empty entity_type
print("\n  [PF4] Project create — empty entity_type")
req("POST", f"/campaigns/{campaign_id}/projects", {
    "name": "Test", "entity_type": "", "entity_name": "Test"
}, 400, "FAIL: project empty entity_type")

# PF5: Project update — invalid status
print("\n  [PF5] Project update — invalid status")
req("PUT", f"/projects/{project_ids[0]}", {"status": "INVALID_STATUS"}, 400, "FAIL: project invalid status")

# PF6: Project update — invalid entity_type
print("\n  [PF6] Project update — invalid entity_type")
req("PUT", f"/projects/{project_ids[0]}", {"entity_type": "spaceship"}, 400, "FAIL: project invalid entity_type")

# --- Summary ---
print("\n" + "=" * 60)
print(f"RESULTS: {passed} passed, {failed} failed, {passed + failed} total")
print("=" * 60)
sys.exit(0 if failed == 0 else 1)
