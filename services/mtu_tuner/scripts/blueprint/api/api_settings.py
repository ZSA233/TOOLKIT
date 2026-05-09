from api_blueprint.includes import *
from blueprint.app import bp


class TestTarget(Model):
    name = String(description="target name")
    url = String(description="target url")
    enabled = Bool(description="whether target is enabled")
    profiles = Array[String](description="profiles using this target")
    order = Int(description="target order")


class SavedSettings(Model):
    version = Int(description="config schema version")
    route_probe = String(description="route probe")
    fallback_probe = String(description="fallback route probe")
    http_proxy = String(description="http proxy")
    clash_api = String(description="clash api")
    proxy_group = String(description="proxy group")
    config_path = String(description="clash config path")
    browser_path = String(description="browser path")
    test_profile = String(description="test profile")
    test_targets = Array[TestTarget](description="configured test targets")
    sweep_mtus = String(description="comma separated sweep mtus")
    target_mtu = Int(description="default target mtu")


class CurrentSettingsRequest(Model):
    settings = SavedSettings(description="settings")


with bp.group("/settings") as views:
    views.GET(
        "/current",
        summary="Load saved settings",
        operation_id="GetCurrentSettings",
    ).RSP(SavedSettings)

    views.PUT(
        "/current",
        summary="Save settings",
        operation_id="SaveCurrentSettings",
    ).REQ(CurrentSettingsRequest).RSP(SavedSettings)
