from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy import inspect
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.database import engine, get_db
from app.models import Operator
from app.schemas import OperatorBase, OperatorRead

router = APIRouter(prefix="/api/operators", tags=["operators"])


def _ensure_operators_table():
    if not inspect(engine).has_table("operators"):
        Operator.__table__.create(bind=engine, checkfirst=True)


def _to_read(op: Operator) -> OperatorRead:
    return OperatorRead(
        id=op.id,
        display_name=op.display_name,
        active=bool(op.active),
    )


@router.get("", response_model=list[OperatorRead])
def list_operators(active_only: bool = True, db: Session = Depends(get_db)):
    _ensure_operators_table()
    q = db.query(Operator)
    if active_only:
        q = q.filter(Operator.active == True)  # noqa: E712
    rows = q.order_by(Operator.display_name).all()
    return [_to_read(op) for op in rows]


@router.post("", response_model=OperatorRead)
def create_operator(data: OperatorBase, db: Session = Depends(get_db)):
    _ensure_operators_table()
    name = (data.display_name or "").strip()
    if not name:
        raise HTTPException(status_code=400, detail="Name required")

    existing = db.query(Operator).filter(Operator.display_name == name).first()
    if existing:
        return _to_read(existing)

    op = Operator(display_name=name, active=True)
    db.add(op)
    try:
        db.commit()
        db.refresh(op)
    except IntegrityError:
        db.rollback()
        existing = db.query(Operator).filter(Operator.display_name == name).first()
        if existing:
            return _to_read(existing)
        raise HTTPException(status_code=400, detail="Duplicate operator name") from None
    except Exception as e:
        db.rollback()
        raise HTTPException(status_code=500, detail=str(e)) from e

    return _to_read(op)
