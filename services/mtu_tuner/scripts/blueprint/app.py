from api_blueprint.includes import *


bp = Blueprint(
    root="/api",
    tags=["mtu-tuner"],
    providers=[
        provider.Req(),
        provider.Handle(),
        provider.Rsp(),
    ],
)

from blueprint.api import api_network
from blueprint.api import api_settings
from blueprint.api import api_system
from blueprint.api import api_tasks
