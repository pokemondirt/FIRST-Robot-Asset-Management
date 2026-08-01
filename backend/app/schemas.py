from datetime import datetime

from typing import Literal



from pydantic import BaseModel, ConfigDict, Field





ProgramType = Literal["FRC", "FTC", "BOTH"]

TrackModeType = Literal["SNP", "BLK"]

TransactionTypeLiteral = Literal["IN", "OUT", "RETURN", "ADJUST"]





class LocationBase(BaseModel):

    code: str

    name_zh: str = ""

    name_en: str = ""





class LocationRead(LocationBase):

    id: int



    class Config:

        from_attributes = True





class OperatorBase(BaseModel):

    display_name: str





class OperatorRead(OperatorBase):
    model_config = ConfigDict(from_attributes=True)

    id: int
    active: bool = True





class ItemBase(BaseModel):

    program: ProgramType = "BOTH"

    track_mode: TrackModeType = "BLK"

    name_zh: str

    name_en: str

    category: str = "MISC"

    spec: str = ""

    unit: str = "pcs"

    min_stock: int = 0

    location_id: int | None = None

    note: str = ""

    active: bool = True





class ItemCreate(ItemBase):

    barcode: str | None = None

    quantity_initial: int = 0





class ItemUpdate(BaseModel):

    program: ProgramType | None = None

    name_zh: str | None = None

    name_en: str | None = None

    category: str | None = None

    spec: str | None = None

    unit: str | None = None

    min_stock: int | None = None

    location_id: int | None = None

    note: str | None = None

    active: bool | None = None





class ItemRead(ItemBase):

    id: int

    barcode: str

    quantity: int = 0

    location_code: str | None = None

    created_at: datetime



    class Config:

        from_attributes = True





class ItemListResponse(BaseModel):

    items: list[ItemRead]

    total: int

    page: int

    page_size: int





class ScanLookupResponse(BaseModel):

    kind: Literal["item", "location", "unknown"]

    item: ItemRead | None = None

    location: LocationRead | None = None





class TransactionCreate(BaseModel):

    barcode: str

    type: TransactionTypeLiteral

    quantity: int = Field(default=1, ge=0)

    operator_id: int | None = None

    note: str = ""





class TransactionRead(BaseModel):

    id: int

    type: str

    item_id: int

    barcode: str

    name_zh: str

    name_en: str

    quantity: int

    operator_name: str | None

    note: str

    created_at: datetime



    class Config:

        from_attributes = True





class TransactionListResponse(BaseModel):

    transactions: list[TransactionRead]

    total: int

    page: int

    page_size: int





class SettingsRead(BaseModel):

    label_width_mm: float

    label_height_mm: float





class SettingsUpdate(BaseModel):

    label_width_mm: float | None = None

    label_height_mm: float | None = None





class StatsResponse(BaseModel):

    total_items: int

    low_stock_count: int

    total_quantity: int





class InboundSummary(BaseModel):

    today_transactions: int

    today_quantity: int

    week_quantity: int





class RecentInboundLine(BaseModel):

    id: int

    barcode: str

    name_zh: str

    name_en: str

    quantity: int

    operator_name: str | None

    created_at: datetime





class LowStockAlert(BaseModel):

    id: int

    barcode: str

    name_zh: str

    name_en: str

    program: str

    category: str

    quantity: int

    min_stock: int

    shortage: int

    unit: str

    level: Literal["warning", "critical"]





class DashboardResponse(BaseModel):

    generated_at: datetime

    inbound: InboundSummary

    recent_inbound: list[RecentInboundLine]

    low_stock_alerts: list[LowStockAlert]

    alert_count: int


