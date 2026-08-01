from sqlalchemy.orm import Session, joinedload

from app.models import Item, Location, Stock
from app.schemas import ItemCreate, ItemRead, ItemUpdate
from app.services.barcode import next_barcode


def item_to_read(item: Item) -> ItemRead:
    qty = item.stock.quantity if item.stock else 0
    loc_code = item.location.code if item.location else None
    return ItemRead(
        id=item.id,
        barcode=item.barcode,
        program=item.program,
        track_mode=item.track_mode,
        name_zh=item.name_zh,
        name_en=item.name_en,
        category=item.category,
        spec=item.spec,
        unit=item.unit,
        min_stock=item.min_stock,
        location_id=item.location_id,
        note=item.note,
        active=item.active,
        quantity=qty,
        location_code=loc_code,
        created_at=item.created_at,
    )


def get_item_by_barcode(db: Session, barcode: str) -> Item | None:
    return (
        db.query(Item)
        .options(joinedload(Item.stock), joinedload(Item.location))
        .filter(Item.barcode == barcode.strip())
        .first()
    )


def create_item(db: Session, data: ItemCreate) -> Item:
    barcode = data.barcode or next_barcode(db, data.program, data.track_mode)
    existing = db.query(Item).filter(Item.barcode == barcode).first()
    if existing:
        raise ValueError(f"Barcode already exists: {barcode}")

    item = Item(
        barcode=barcode,
        program=data.program,
        track_mode=data.track_mode,
        name_zh=data.name_zh,
        name_en=data.name_en,
        category=data.category,
        spec=data.spec,
        unit=data.unit,
        min_stock=data.min_stock,
        location_id=data.location_id,
        note=data.note,
        active=data.active,
    )
    db.add(item)
    db.flush()
    stock = Stock(item_id=item.id, quantity=max(0, data.quantity_initial))
    db.add(stock)
    db.commit()
    db.refresh(item)
    return (
        db.query(Item)
        .options(joinedload(Item.stock), joinedload(Item.location))
        .filter(Item.id == item.id)
        .one()
    )


def update_item(db: Session, item: Item, data: ItemUpdate) -> Item:
    payload = data.model_dump(exclude_unset=True)
    for key, value in payload.items():
        setattr(item, key, value)
    db.commit()
    db.refresh(item)
    return (
        db.query(Item)
        .options(joinedload(Item.stock), joinedload(Item.location))
        .filter(Item.id == item.id)
        .one()
    )
