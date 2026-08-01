from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from app.database import get_db
from app.models import Location
from app.schemas import LocationBase, LocationRead

router = APIRouter(prefix="/api/locations", tags=["locations"])


@router.get("", response_model=list[LocationRead])
def list_locations(db: Session = Depends(get_db)):
    return db.query(Location).order_by(Location.code).all()


@router.post("", response_model=LocationRead)
def create_location(data: LocationBase, db: Session = Depends(get_db)):
    code = data.code.strip()
    if db.query(Location).filter(Location.code == code).first():
        raise HTTPException(400, "Location code exists")
    loc = Location(code=code, name_zh=data.name_zh, name_en=data.name_en)
    db.add(loc)
    db.commit()
    db.refresh(loc)
    return loc
