from fastapi import APIRouter, Depends, HTTPException, Query
from fastapi.responses import Response
from sqlalchemy.orm import Session

from app.database import get_db
from app.models import Item
from app.services.labels_pdf import build_labels_pdf

router = APIRouter(prefix="/api/labels", tags=["labels"])


@router.get("/pdf")
def labels_pdf(
    ids: str = Query(..., description="Comma-separated item IDs"),
    db: Session = Depends(get_db),
):
    id_list = [int(x) for x in ids.split(",") if x.strip().isdigit()]
    if not id_list:
        raise HTTPException(400, "No valid item IDs")
    items = (
        db.query(Item)
        .filter(Item.id.in_(id_list), Item.active.is_(True))
        .order_by(Item.barcode)
        .all()
    )
    if not items:
        raise HTTPException(404, "No items found")
    pdf = build_labels_pdf(items)
    return Response(
        content=pdf,
        media_type="application/pdf",
        headers={"Content-Disposition": "inline; filename=labels.pdf"},
    )
