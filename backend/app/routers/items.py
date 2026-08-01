from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy import or_
from sqlalchemy.orm import Session, joinedload

from app.database import get_db
from app.models import Item, Location, Stock
from app.schemas import ItemCreate, ItemListResponse, ItemRead, ItemUpdate, ScanLookupResponse
from app.services import items as item_service
from app.services.barcode import next_barcode

router = APIRouter(prefix="/api/items", tags=["items"])


@router.get("", response_model=ItemListResponse)
def list_items(
    q: str | None = None,
    program: str | None = None,
    category: str | None = None,
    track_mode: str | None = None,
    active_only: bool = True,
    page: int = Query(1, ge=1),
    page_size: int = Query(50, ge=1, le=200),
    db: Session = Depends(get_db),
):
    query = db.query(Item)
    if active_only:
        query = query.filter(Item.active.is_(True))
    if program:
        query = query.filter(Item.program == program)
    if category:
        query = query.filter(Item.category == category)
    if track_mode:
        query = query.filter(Item.track_mode == track_mode)
    if q:
        like = f"%{q}%"
        query = query.filter(
            or_(
                Item.barcode.ilike(like),
                Item.name_zh.ilike(like),
                Item.name_en.ilike(like),
            )
        )
    total = query.count()
    rows = (
        query.options(
            joinedload(Item.stock), joinedload(Item.location)
        )
        .order_by(Item.id.desc())
        .offset((page - 1) * page_size)
        .limit(page_size)
        .all()
    )
    return ItemListResponse(
        items=[item_service.item_to_read(i) for i in rows],
        total=total,
        page=page,
        page_size=page_size,
    )


@router.get("/lookup/{code}", response_model=ScanLookupResponse)
def lookup_code(code: str, db: Session = Depends(get_db)):
    code = code.strip()
    item = item_service.get_item_by_barcode(db, code)
    if item:
        return ScanLookupResponse(
            kind="item", item=item_service.item_to_read(item), location=None
        )
    loc = db.query(Location).filter(Location.code == code).first()
    if loc:
        from app.schemas import LocationRead

        return ScanLookupResponse(
            kind="location",
            item=None,
            location=LocationRead.model_validate(loc),
        )
    return ScanLookupResponse(kind="unknown", item=None, location=None)


@router.get("/{item_id}", response_model=ItemRead)
def get_item(item_id: int, db: Session = Depends(get_db)):
    item = (
        db.query(Item)
        .options(joinedload(Item.stock), joinedload(Item.location))
        .filter(Item.id == item_id)
        .first()
    )
    if not item:
        raise HTTPException(404, "Item not found")
    return item_service.item_to_read(item)


@router.post("", response_model=ItemRead)
def create_item(data: ItemCreate, db: Session = Depends(get_db)):
    try:
        item = item_service.create_item(db, data)
    except ValueError as e:
        raise HTTPException(400, str(e)) from e
    return item_service.item_to_read(item)


@router.post("/next-barcode")
def preview_next_barcode(
    program: str = "BOTH",
    track_mode: str = "BLK",
    db: Session = Depends(get_db),
):
    return {"barcode": next_barcode(db, program, track_mode)}


@router.patch("/{item_id}", response_model=ItemRead)
def patch_item(
    item_id: int, data: ItemUpdate, db: Session = Depends(get_db)
):
    item = db.query(Item).filter(Item.id == item_id).first()
    if not item:
        raise HTTPException(404, "Item not found")
    item = item_service.update_item(db, item, data)
    return item_service.item_to_read(item)


@router.delete("/{item_id}")
def delete_item(item_id: int, db: Session = Depends(get_db)):
    item = db.query(Item).filter(Item.id == item_id).first()
    if not item:
        raise HTTPException(404, "Item not found")
    item.active = False
    db.commit()
    return {"ok": True}
