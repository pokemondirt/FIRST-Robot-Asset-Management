from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.orm import Session, joinedload

from app.database import get_db
from app.models import Item, Transaction
from app.schemas import TransactionCreate, TransactionListResponse, TransactionRead
from app.services import transactions as tx_service

router = APIRouter(prefix="/api/transactions", tags=["transactions"])


def _tx_to_read(tx: Transaction) -> TransactionRead:
    item = tx.item
    op_name = tx.operator.display_name if tx.operator else None
    return TransactionRead(
        id=tx.id,
        type=tx.type,
        item_id=tx.item_id,
        barcode=item.barcode,
        name_zh=item.name_zh,
        name_en=item.name_en,
        quantity=tx.quantity,
        operator_name=op_name,
        note=tx.note,
        created_at=tx.created_at,
    )


@router.post("", response_model=TransactionRead)
def create_transaction(data: TransactionCreate, db: Session = Depends(get_db)):
    try:
        tx = tx_service.apply_transaction(db, data)
    except tx_service.TransactionError as e:
        raise HTTPException(
            status_code=400, detail={"code": e.code, "message": e.message}
        ) from e
    tx = (
        db.query(Transaction)
        .options(
            joinedload(Transaction.item),
            joinedload(Transaction.operator),
        )
        .filter(Transaction.id == tx.id)
        .one()
    )
    return _tx_to_read(tx)


@router.get("", response_model=TransactionListResponse)
def list_transactions(
    page: int = Query(1, ge=1),
    page_size: int = Query(50, ge=1, le=200),
    item_id: int | None = None,
    db: Session = Depends(get_db),
):
    query = db.query(Transaction).options(
        joinedload(Transaction.item),
        joinedload(Transaction.operator),
    )
    if item_id:
        query = query.filter(Transaction.item_id == item_id)
    total = query.count()
    rows = (
        query.order_by(Transaction.created_at.desc())
        .offset((page - 1) * page_size)
        .limit(page_size)
        .all()
    )
    return TransactionListResponse(
        transactions=[_tx_to_read(t) for t in rows],
        total=total,
        page=page,
        page_size=page_size,
    )
